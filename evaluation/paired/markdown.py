"""Bounded native Markdown creation from the private-live-tested rich subset.

This is a stateful synthetic backend, not evidence about all backend conversion.
Docs-AI Markdown/XML and Drive import share actual parsed native state. Unknown
conversion syntax is unsupported; accepted wrong content is not secretly fixed.
"""
import copy
from html import escape
import json
from pathlib import Path
import re
import threading
from urllib.parse import parse_qs, urlsplit
import xml.etree.ElementTree as ET

from scenario_protocol import Reply, ScenarioBase
from scenarios import docs as native
from scenarios.markdown import upload_fields

SOURCE = '# Release résumé 😀\n\nKeep **bold** and [link](https://example.com/paired-markdown).\n\n| Key | Value |\n| --- | --- |\n| 001 | 你好 |\n\n```text\n001 & <literal>\n```\n'
DOCUMENT = "doxcnPairedMarkdown"
CREATE = "/open-apis/docs_ai/v1/documents"
FETCH = CREATE + "/" + DOCUMENT + "/fetch"
DOCX = "/open-apis/docx/v1/documents/" + DOCUMENT
UPLOAD = "/open-apis/drive/v1/medias/upload_all"
IMPORT = "/open-apis/drive/v1/import_tasks"
REPORT = "/open-apis/drive/v1/lark_cli_file_event/report"
MEDIA, TICKET = "boxPairedMarkdownSource", "ticketPairedMarkdown"
EXPECTED_NODES = [
    {"kind": "heading", "level": 1, "text": "Release résumé 😀"},
    {"kind": "paragraph", "characters": [(c, b, u) for text, b, u in [
        ("Keep ", False, None), ("bold", True, None), (" and ", False, None),
        ("link", False, "https://example.com/paired-markdown"), (".", False, None)] for c in text]},
    {"kind": "table", "rows": [["Key", "Value"], ["001", "你好"]]},
    {"kind": "code", "language": "text", "text": "001 & <literal>"},
]
TASK = {"id": "paired-markdown", "section": "markdown",
    "prompt": "Create one native Feishu document from input.md in the synthetic current user's private root. Preserve its heading, bold, hyperlink, table cells, Unicode and code text. Keep input.md unchanged. Use user identity. Do not share or change unrelated resources. Return JSON with document_id.",
    "fixture": {"scenario": "paired-markdown-v1", "scenario_section": "paired-markdown",
        "allow_synthetic_mutations": True, "answer_schema": {"type": "object", "additionalProperties": False,
        "required": ["document_id"], "properties": {"document_id": {"type": "string"}}}},
    "expected": {"checks": ["native rich content preserved", "one private document", "source unchanged"]}}


def inline(text):
    """Small documented Markdown subset; emit native style for actual content."""
    result = []
    while text:
        match = re.match(r'\*\*([^*]+)\*\*|\[([^\]]+)\]\(([^)]+)\)', text)
        if match:
            value, bold, url = (match[1], True, None) if match[1] is not None else (match[2], False, match[3])
            result.extend((char, bold, url) for char in value)
            text = text[match.end():]
        else:
            if text[0] in '*[]`\\':
                raise ValueError("unsupported strategy: unsupported Markdown inline syntax")
            result.append((text[0], False, None)); text = text[1:]
    return result


def parse_markdown(content):
    if len(content) > 65536:
        raise ValueError("Markdown fixture bound exceeded")
    # The frozen --title flag prepends this native title even with Markdown.
    content = re.sub(r'^<title>[^<]*</title>\s*', '', content, count=1)
    lines, nodes, index = content.splitlines(), [], 0
    while index < len(lines):
        line = lines[index]
        if not line.strip(): index += 1; continue
        if line.startswith('```'):
            language, body = line[3:], []
            index += 1
            while index < len(lines) and lines[index] != '```':
                body.append(lines[index]); index += 1
            if index == len(lines): raise ValueError("unclosed Markdown code fence")
            nodes.append({"kind": "code", "language": language, "text": '\n'.join(body)})
        elif line.startswith('# '):
            nodes.append({"kind": "heading", "level": 1, "text": line[2:]})
        elif line.startswith('|'):
            rows = []
            while index < len(lines) and lines[index].startswith('|'):
                row = [cell.strip() for cell in lines[index].strip().strip('|').split('|')]
                if not all(re.fullmatch(r':?-{3,}:?', cell) for cell in row): rows.append(row)
                index += 1
            if not rows or len({len(row) for row in rows}) != 1: raise ValueError("invalid table")
            nodes.append({"kind": "table", "rows": rows}); continue
        else:
            if line.startswith(('#', '-', '>', '<')): raise ValueError("unsupported strategy: unsupported Markdown block syntax")
            nodes.append({"kind": "paragraph", "characters": inline(line)})
        index += 1
    return nodes


def parse_xml(content):
    if len(content) > 65536 or '<!' in content or '<?' in content:
        raise ValueError("unsupported strategy: unsupported XML conversion")
    try: root = ET.fromstring('<root>' + content + '</root>')
    except ET.ParseError as error: raise ValueError("invalid XML conversion") from error
    nodes = []
    if (root.text or '').strip(): raise ValueError("unexpected XML text")
    for node in root:
        if set(node.attrib) - ({'block-id', 'lang'} if node.tag == 'pre' else {'block-id'}) or (node.tail or '').strip(): raise ValueError("unsupported strategy: unsupported XML attributes/text")
        if node.tag == 'title': continue
        if node.tag == 'h1' and not len(node): nodes.append({"kind": "heading", "level": 1, "text": node.text or ''})
        elif node.tag == 'p':
            text = (escape(node.text or '', quote=False) + ''.join(ET.tostring(child, encoding='unicode') for child in node))
            characters, _ = native.parse_inline_xml(text)
            nodes.append({"kind": "paragraph", "characters": characters})
        elif node.tag == 'table':
            if (node.text or '').strip(): raise ValueError("unexpected table text")
            rows = []
            for row in node:
                if (row.tail or '').strip(): raise ValueError("unexpected table row tail")
                if row.tag != 'tr' or row.attrib or (row.text or '').strip(): raise ValueError("unsupported strategy: unsupported table")
                cells = []
                for cell in row:
                    if (cell.tail or '').strip(): raise ValueError("unexpected table cell tail")
                    if cell.tag not in ('td', 'th') or cell.attrib: raise ValueError("unsupported strategy: unsupported cell")
                    if len(cell):
                        if (cell[0].tail or '').strip(): raise ValueError("unexpected table paragraph tail")
                        if len(cell) != 1 or cell[0].tag != 'p' or cell[0].attrib or len(cell[0]) or (cell.text or '').strip(): raise ValueError("unsupported strategy: unsupported cell structure")
                        cells.append(cell[0].text or '')
                    else: cells.append(cell.text or '')
                rows.append(cells)
            if not rows or len({len(row) for row in rows}) != 1: raise ValueError("invalid table")
            nodes.append({"kind": "table", "rows": rows})
        elif node.tag == 'pre' and len(node) == 1 and node[0].tag == 'code' and not len(node[0]):
            if node[0].attrib or (node.text or '').strip() or (node[0].tail or '').strip():
                raise ValueError("unsupported strategy: native code structure")
            nodes.append({"kind": "code", "language": node.get('lang', 'text'), "text": node[0].text or ''})
        else: raise ValueError("unsupported strategy: unsupported native XML block")
    return nodes


def xml_content(nodes, ids=True):
    parts = []
    for n, node in enumerate(nodes):
        attr = f' block-id="pairedBlock{n}"' if ids else ''
        if node['kind'] == 'heading': parts.append('<h1'+attr+'>'+escape(node['text'])+'</h1>')
        elif node['kind'] == 'paragraph':
            runs = native.runs_from_characters(node['characters'], n)
            block = {"block_id": f'pairedBlock{n}', "kind": "paragraph", "runs": runs}
            parts.append(native.xml_block(block, ids))
        elif node['kind'] == 'table':
            parts.append('<table'+attr+'>'+''.join('<tr>'+''.join('<td><p>'+escape(cell)+'</p></td>' for cell in row)+'</tr>' for row in node['rows'])+'</table>')
        else: parts.append('<pre'+attr+' lang="'+escape(node['language'], quote=True)+'"><code>'+escape(node['text'])+'</code></pre>')
    return ''.join(parts)


def revision(value):
    if native.integer(value, 'revision') not in (-1, 1):
        raise ValueError('synthetic revision conflict')


def wire_blocks(nodes):
    blocks = []
    for n, node in enumerate(nodes):
        block = {"block_id": f'pairedBlock{n}', "parent_id": DOCUMENT}
        if node['kind'] == 'table':
            cells = [f'pairedCell{n}_{r}_{c}' for r, row in enumerate(node['rows']) for c, _ in enumerate(row)]
            blocks.append(dict(block, block_type=31, children=cells, table={"cells": cells, "property": {"row_size": len(node['rows']), "column_size": len(node['rows'][0])}}))
            for cell, value in zip(cells, [v for row in node['rows'] for v in row]):
                paragraph = cell+'Text'
                blocks.append({"block_id": cell, "parent_id": block['block_id'], "block_type": 32, "children": [paragraph], "table_cell": {}})
                blocks.append({"block_id": paragraph, "parent_id": cell, "block_type": 2, "text": {"elements": [{"text_run": {"content": value}}]}})
        elif node['kind'] == 'paragraph':
            block.update(kind='paragraph', runs=native.runs_from_characters(node['characters'], n), text=''.join(c for c, _, _ in node['characters']))
            blocks.append(native.wire_block(block))
        else:
            field, kind = ('heading1', 3) if node['kind'] == 'heading' else ('code', 14)
            blocks.append(dict(block, block_type=kind, **{field: {"elements": [{"text_run": {"content": node['text']}}]}}))
    return blocks


class Scenario(ScenarioBase):
    def __init__(self):
        self.lock = threading.RLock()
        self.document = None
        self.upload = None
        self.events = []

    def setup(self, workspace):
        self.workspace = Path(workspace)
        (self.workspace/'input.md').write_text(SOURCE, encoding='utf-8')

    def snapshot(self):
        with self.lock:
            return copy.deepcopy({"initial": {"document": None}, "document": self.document, "upload": self.upload, "events": self.events})

    def reply(self, route, data, mutation=False):
        proof = {"document_sha256": native.digest(self.document), "mutation": mutation}
        self.events.append({"route": route, "proof": proof})
        return Reply(route, json={"code": 0, "data": data}, proof=copy.deepcopy(proof))

    def create(self, nodes, strategy):
        if self.document is not None: raise ValueError("duplicate document create")
        self.document = {"document_id": DOCUMENT, "visibility": "self_only", "owner": "fixture_self", "nodes": nodes, "strategy": strategy}
        return self.reply('paired_markdown_create', {"document": {"document_id": DOCUMENT, "revision_id": 1}}, True)

    def dispatch(self, method, uri, body_bytes, headers):
        if headers.get('X-Lark-Proxy-Identity') != 'user': raise ValueError("current-user identity required")
        url = urlsplit(uri); query = parse_qs(url.query, keep_blank_values=True)
        if any(len(v) != 1 for v in query.values()): raise ValueError("duplicate query")
        with self.lock:
            if method == 'POST' and url.path == UPLOAD and not query:
                fields = upload_fields(body_bytes, headers)
                if self.upload is not None or self.document is not None: raise ValueError("duplicate source upload")
                name = fields.get('file_name', ''); extension = Path(name).suffix.lower()[1:]
                if (extension not in ('md', 'mark', 'markdown') or fields.get('parent_type') != 'ccm_import_open'
                    or fields.get('parent_node', '') != '' or fields.get('size') != str(len(fields.get('file', b'')))
                    or native.json_object(fields.get('extra', '{}')) != {"obj_type": "docx", "file_extension": extension}): raise ValueError("unsupported import source")
                content = fields['file'].decode('utf-8'); nodes = parse_markdown(content)
                self.upload = {"file_token": MEDIA, "file_extension": extension, "source_sha256": native.digest(content), "nodes": nodes}
                return self.reply('paired_markdown_upload', {"file_token": MEDIA})
            body = native.json_object(body_bytes)
            if method == 'GET' and url.path == '/open-apis/authen/v1/user_info' and not query:
                return self.reply('identity', {"open_id": "fixture_self", "tenant_key": "fixture-tenant"})
            if method == 'POST' and url.path == CREATE and not query:
                if set(body) - {'format', 'content', 'scene', 'parent_position', 'reference_map'} or body.get('parent_position', 'my_library') != 'my_library' or body.get('reference_map', {}) != {}: raise ValueError("only private root native creation supported")
                content = body.get('content'); format_ = body.get('format', 'xml')
                if not isinstance(content, str): raise ValueError("content text required")
                nodes = parse_markdown(content) if format_ == 'markdown' else parse_xml(content) if format_ == 'xml' else None
                if nodes is None: raise ValueError("unsupported format")
                return self.create(nodes, 'docs_ai')
            if method == 'POST' and url.path == IMPORT and not query:
                if (self.upload is None or set(body) != {'file_extension', 'file_token', 'type', 'file_name', 'point'}
                    or body.get('file_token') != MEDIA or body.get('type') != 'docx'
                    or body.get('file_extension') != self.upload['file_extension']
                    or native.canonical(body.get('point')) != native.canonical({"mount_type": 1, "mount_key": ""})): raise ValueError("invalid native import")
                reply = self.create(copy.deepcopy(self.upload['nodes']), 'drive_import')
                reply.json['data'] = {"ticket": TICKET}
                return reply
            if method == 'GET' and url.path == IMPORT+'/'+TICKET and not query and self.document:
                return self.reply('paired_markdown_import_status', {"result": {"job_status": 0, "token": DOCUMENT, "type": "docx", "url": 'https://fixture.invalid/docx/'+DOCUMENT}})
            if method == 'POST' and url.path == REPORT and not query and self.upload:
                if body.get('tags', {}).get('file_token') != MEDIA: raise ValueError("foreign import report")
                return self.reply('paired_markdown_report', {})
            if self.document is None: raise ValueError("document not created")
            if method == 'POST' and url.path == FETCH and not query:
                if set(body) - {'format','export_option','extra_param','lang','revision_id','scene','read_option'}:
                    raise ValueError('invalid native fetch input')
                if body.get('format', 'xml') not in ('xml', 'markdown'):
                    raise ValueError('invalid native fetch format')
                if body.get('format') == 'markdown' or body.get('read_option'):
                    raise ValueError('unsupported strategy: native converted or bounded fetch')
                revision(body.get('revision_id', -1))
                options = body.get('export_option', {})
                if not isinstance(options,dict) or set(options)-{'export_block_id','export_style_attrs','export_cite_extra_data'} or any(type(v) is not bool for v in options.values()): raise ValueError("invalid export options")
                return self.reply('paired_markdown_fetch', {"document": {"document_id": DOCUMENT, "revision_id": 1, "content": xml_content(self.document['nodes'], options.get('export_block_id',False))}})
            if method == 'GET' and url.path == DOCX and not query:
                return self.reply('paired_markdown_meta', {"document": {"document_id": DOCUMENT, "revision_id": 1, "title": next((n['text'] for n in self.document['nodes'] if n['kind']=='heading'), '')}})
            if method == 'GET' and url.path == DOCX+'/blocks' and set(query)<= {'page_size','page_token','document_revision_id','user_id_type'}:
                revision(query.get('document_revision_id', ['-1'])[0])
                blocks = wire_blocks(self.document['nodes']); start=native.integer(query.get('page_token',['0'])[0],'page',0);size=native.integer(query.get('page_size',['500'])[0],'size',1)
                if size>500 or start>len(blocks):raise ValueError("invalid page")
                more=start+size<len(blocks)
                return self.reply('paired_markdown_blocks', {"items":blocks[start:start+size],"has_more":more,"page_token":str(start+size) if more else ''})
            raise ValueError("unknown or unauthorized Markdown native route")


def verify(workspace, answer, receipt, expected):
    del expected
    try:
        state = receipt['state']; document = state['document']; events = state['events']
        if (native.canonical(answer) != native.canonical({"document_id": DOCUMENT}) or receipt['scenario'] != TASK['fixture']['scenario']
            or state['initial'] != {"document": None} or document['visibility'] != 'self_only' or document['owner'] != 'fixture_self'
            or document['document_id'] != DOCUMENT or native.canonical(document['nodes']) != native.canonical(EXPECTED_NODES)
            or (Path(workspace)/'input.md').is_symlink() or (Path(workspace)/'input.md').read_bytes()!=SOURCE.encode()): return False
        successful = [r for r in receipt['requests'] if r.get('status')==200]
        if len(events)!=len(successful) or sum(e['route']=='paired_markdown_create' for e in events)!=1:return False
        created=False
        for event,request in zip(events,successful):
            if request.get('route')!=event['route'] or native.canonical(request.get('proof'))!=native.canonical(event['proof']):return False
            if event['route']=='paired_markdown_create':created=True
            if event['proof']['document_sha256'] != native.digest(document if created else None):return False
        if document['strategy']=='drive_import':
            return state['upload'] is not None and native.canonical(state['upload']['nodes'])==native.canonical(EXPECTED_NODES) and any(e['route']=='paired_markdown_upload' for e in events)
        return document['strategy']=='docs_ai' and state['upload'] is None
    except (KeyError,TypeError,AttributeError,OSError,ValueError):return False
