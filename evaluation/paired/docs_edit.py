"""Paired native Docs edit, with live-observed replacement-span semantics.

The old development fixture remains untouched. Its rendering, routing, typed
native PATCH, paragraph replacement and independent verifier are reused here.
"""
import copy
from html import escape
import re
import xml.etree.ElementTree as ET

from scenarios import docs as native
from paired.docs_revision import reject_unmodeled_history

TASK = copy.deepcopy(native.TASK)
TASK.update(id="paired-docs-edit", section="docs")
TASK["fixture"].update(scenario="paired-docs-edit-v1", scenario_section="paired-docs-edit")
TASK["prompt"] = (
    f"In document {native.DOCUMENT}, change only the exact phrase Old status to New status. "
    "Preserve native bold formatting and the existing hyperlink on status, the Unicode "
    "text, and every other block/resource. Do not create, share or delete resources. "
    "Return JSON with document_id and paragraph_text (the complete changed paragraph)."
)


def markdown_inline_xml(content):
    """Bounded native-checked paragraph syntax; unsupported marks are not flattened."""
    if not isinstance(content, str) or len(content) > 65536 or '\n' in content or '\r' in content:
        raise ValueError(native.UNSUPPORTED_XML)
    result = []
    while content:
        match = re.match(r'\*\*\[([^\[\]\*]+)\]\(([^()\s]+)\)\*\*', content)
        if match:
            if any(c in match[1] for c in '_`<>\\'):
                raise ValueError(native.UNSUPPORTED_XML)
            result.append('<b><a href="'+escape(match[2],quote=True)+'">'+escape(match[1],quote=False)+'</a></b>')
        else:
            match = re.match(r'\*\*([^*]+)\*\*|\[([^\[\]]+)\]\(([^()\s]+)\)',content)
            if match:
                if any(c in (match[1] if match[1] is not None else match[2]) for c in '*_`[]<>\\'):
                    raise ValueError(native.UNSUPPORTED_XML)
                result.append('<b>'+escape(match[1],quote=False)+'</b>' if match[1] is not None else
                              '<a href="'+escape(match[3],quote=True)+'">'+escape(match[2],quote=False)+'</a>')
            else:
                if content[0] in '*_`[]<>\\':
                    raise ValueError(native.UNSUPPORTED_XML)
                result.append(escape(content[0],quote=False))
                content=content[1:]
                continue
        content=content[match.end():]
    return ''.join(result)


def xml_paragraph(content):
    """Accept the observed native id attribute and bare inline paragraph fragments."""
    if not isinstance(content,str) or len(content)>65536 or '<!' in content or '<?' in content:
        raise ValueError(native.UNSUPPORTED_XML)
    try:
        root=ET.fromstring('<fragment>'+content+'</fragment>')
    except ET.ParseError as error:
        raise ValueError(native.UNSUPPORTED_XML) from error
    if len(root)==1 and root[0].tag=='p' and not (root.text or '').strip() and not (root[0].tail or '').strip():
        paragraph=root[0]
        if set(paragraph.attrib)-{'id','block-id'}:
            raise ValueError(native.UNSUPPORTED_XML)
        paragraph.attrib.pop('id',None)
        content=ET.tostring(paragraph,encoding='unicode')
    else:
        native.parse_inline_xml(content)
        content='<p>'+content+'</p>'
    native.parse_inline_xml(content,paragraph=True)
    return content


class Scenario(native.Scenario):
    def dispatch(self, method, uri, body_bytes, headers):
        with self.lock:
            reject_unmodeled_history(self, method, uri, body_bytes, headers)
            return super().dispatch(method, uri, body_bytes, headers)

    def _fetch(self, data):
        reply = super()._fetch(data)
        reply.json["data"].pop("returned_block_ids", None)
        return reply

    def _replace(self, data):
        if data.get("command") != "str_replace":
            return super()._replace(data)
        data = copy.deepcopy(data)
        content = data.get("content", "")
        if not isinstance(content, str):
            raise ValueError("replacement content must be text")
        format_ = data.get("format", "xml")
        if format_ == "xml":
            # The complete matched span is replaced by the supplied native
            # representation. Plain XML has no marks; unchanged suffix text
            # inside the match does not inherit the old bold/link formatting.
            native.parse_inline_xml(content)
        elif format_ == "markdown":
            # Native Markdown str_replace matches serialized Markdown. A narrow
            # plain-run replacement preserves adjacent marks but rebuilds the block.
            # Rich patterns/replacements need separate observations before emulation.
            pattern=data.get('pattern')
            if not isinstance(pattern,str) or not pattern or any(c in pattern+content for c in "*_`[]<>\\"):
                raise ValueError(native.UNSUPPORTED_XML)
            self._revision(data.get('revision_id',-1))
            if set(data)-{'command','format','pattern','content','revision_id','scene','reference_map'} or data.get('reference_map',{})!={}:
                raise ValueError('unsupported precise edit input')
            matches=[block for block in self.state['blocks'] if block['kind']!='image' and pattern in native.text_of(block)]
            if len(matches)!=1 or matches[0]["text"].count(pattern)!=1:
                raise ValueError('missing or ambiguous literal match')
            block=matches[0]
            if block['kind']!='paragraph':
                raise ValueError(native.UNSUPPORTED_RECONSTRUCTION)
            chars=native.native_characters(block);start=block['text'].index(pattern)
            if any(bold or url for _,bold,url in chars[start:start+len(pattern)]):
                return self._reply('docs_update',{'result':'failed','document':{'revision_id':self.state['revision_id']},
                    'warnings':['degrade_code=1013,msg=str_replace pattern was not found in the document. Verify the pattern exactly matches existing document text (case-sensitive, including whitespace and newlines)']},
                    {'mutation':False,'document_id':native.DOCUMENT})
            chars[start:start+len(pattern)]=[(c,False,None) for c in content]
            replacement={'kind':'paragraph','runs':native.runs_from_characters(chars,self.state['revision_id']+1)}
            return super()._replace_paragraph({'command':'block_replace','format':'xml','block_id':block['block_id'],
                'revision_id':data.get('revision_id',-1),'content':native.xml_block(replacement,False)})
        else:
            raise ValueError("unsupported replacement format")
        data["content"] = "<span>" + content + "</span>"
        return super()._replace(data)

    def _replace_paragraph(self, data):
        data=copy.deepcopy(data)
        if data.get('format','xml')=='markdown':
            data['content']='<p>'+markdown_inline_xml(data.get('content'))+'</p>'
            data['format']='xml'
        elif data.get('format','xml')=='xml':
            data['content']=xml_paragraph(data.get('content'))
        return super()._replace_paragraph(data)


def verify(workspace, answer, receipt, expected):
    if not isinstance(receipt, dict) or receipt.get("scenario") != TASK["fixture"]["scenario"]:
        return False
    adapted = dict(receipt, scenario=native.TASK["fixture"]["scenario"])
    return native.verify(workspace, answer, adapted, expected)


def verification_diagnostics(workspace, answer, receipt, expected):
    """Explain failed native checks without changing acceptance or retaining content."""
    if verify(workspace, answer, receipt, expected):
        return []
    reasons = []
    try:
        if native.canonical(answer) != native.canonical(native.EXPECTED_ANSWER):
            reasons.append('paired_docs_answer_mismatch')
        state = receipt['state']
        if native.canonical(state['initial']) != native.canonical(native.INITIAL):
            reasons.append('paired_docs_receipt_invalid')
        if not state['mutations']:
            reasons.append('paired_docs_mutation_missing')
        if native.preserved_outside_target(state['final']) != native.preserved_outside_target(native.INITIAL):
            reasons.append('paired_docs_unrelated_state_changed')
        desired = copy.deepcopy(native.INITIAL['blocks'][1])
        desired['text'] = native.EXPECTED_ANSWER['paragraph_text']
        desired['runs'][0]['text'] = 'New '
        actual = native.native_characters(state['final']['blocks'][1])
        wanted = native.native_characters(desired)
        if [char for char, _, _ in actual] != [char for char, _, _ in wanted]:
            reasons.append('paired_docs_text_mismatch')
        else:
            if [bold for _, bold, _ in actual] != [bold for _, bold, _ in wanted]:
                reasons.append('paired_docs_bold_mismatch')
            if [url for _, _, url in actual] != [url for _, _, url in wanted]:
                reasons.append('paired_docs_link_mismatch')
    except (KeyError, TypeError, ValueError, IndexError, AttributeError):
        return ['paired_docs_receipt_invalid']
    return sorted(set(reasons)) or ['paired_docs_proof_mismatch']
