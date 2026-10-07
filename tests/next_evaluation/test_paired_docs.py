# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Paired Docs fixtures: no models/live requests, optional archived CLI replay."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0,str(Path(__file__).resolve().parents[2]/'evaluation'))
from paired import docs_edit, docs_read
from scenarios import docs as native


class DocsPairedTests(unittest.TestCase):
    def call(self, module, scenario, records, method, path, body=None):
        reply=scenario.dispatch(method,path,json.dumps(body or {}).encode(),{'X-Lark-Proxy-Identity':'user'})
        records.append({'route':reply.route,'status':reply.status,'proof':copy.deepcopy(reply.proof)})
        return reply.json['data']

    def receipt(self,module,scenario,records):
        return {'scenario':module.TASK['fixture']['scenario'],'state':scenario.snapshot(),'requests':records}

    def test_read_native_fact_and_block_and_no_simple_sidechannel(self):
        s=docs_read.Scenario();r=[]
        data=self.call(docs_read,s,r,'POST',native.AI_PATH+'/fetch',{'format':'xml'})
        self.assertNotIn('returned_block_ids',data)
        self.assertFalse(docs_read.verify(None,docs_read.ANSWER,self.receipt(docs_read,s,r),{}))
        self.call(docs_read,s,r,'POST',native.AI_PATH+'/fetch',{'format':'xml','export_option':{'export_block_id':True}})
        self.assertTrue(docs_read.verify(None,docs_read.ANSWER,self.receipt(docs_read,s,r),{}))
        with self.assertRaises(ValueError):self.call(docs_read,s,r,'PUT',native.AI_PATH,{'command':'str_replace'})
        bad=self.receipt(docs_read,s,r);bad['state']['final']['blocks'][2]['text']='forged'
        self.assertFalse(docs_read.verify(None,docs_read.ANSWER,bad,{}))

    def test_read_blocks_and_wrong_answer_proof(self):
        s=docs_read.Scenario();r=[]
        self.call(docs_read,s,r,'GET',native.DOCX_PATH+'/blocks/'+docs_read.BLOCK)
        receipt=self.receipt(docs_read,s,r)
        self.assertTrue(docs_read.verify(None,docs_read.ANSWER,receipt,{}))
        self.assertFalse(docs_read.verify(None,dict(docs_read.ANSWER,approval_code='wrong'),receipt,{}))
        receipt['requests']=[]
        self.assertFalse(docs_read.verify(None,docs_read.ANSWER,receipt,{}))

    def test_plain_replacement_removes_marks_not_inherits_suffix(self):
        s=docs_edit.Scenario();r=[]
        self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'str_replace','format':'xml','pattern':'Old status','content':'New status'})
        chars=native.native_characters(s.state['blocks'][1])
        self.assertTrue(all(not bold and url is None for _,bold,url in chars))
        self.assertFalse(docs_edit.verify(None,native.EXPECTED_ANSWER,self.receipt(docs_edit,s,r),{}))

    def test_explicit_xml_narrow_plain_and_native_patch_are_equivalent(self):
        for strategy in ('xml','narrow_plain','patch','block_replace'):
            with self.subTest(strategy=strategy):
                s=docs_edit.Scenario();r=[]
                if strategy=='patch':
                    els=native.wire_block(s.state['blocks'][1])['text']['elements'];els[0]['text_run']['content']='New '
                    self.call(docs_edit,s,r,'PATCH',native.DOCX_PATH+'/blocks/'+native.TARGET,{'update_text_elements':{'elements':els}})
                else:
                    data={'command':'str_replace','format':'xml','pattern':'Old status','content':'New <a href="synthetic://reference"><b>status</b></a>'}
                    if strategy=='narrow_plain':data.update(pattern='Old',content='New')
                    if strategy=='block_replace':data={'command':'block_replace','block_id':native.TARGET,'format':'xml','content':'<p>New <b><a href="synthetic://reference">status</a></b> — 你好</p>'}
                    self.call(docs_edit,s,r,'PUT',native.AI_PATH,data)
                receipt=self.receipt(docs_edit,s,r)
                self.assertTrue(docs_edit.verify(None,native.EXPECTED_ANSWER,receipt,{}))
                receipt['state']['final']['blocks'][3]['token']='lost'
                self.assertFalse(docs_edit.verify(None,native.EXPECTED_ANSWER,receipt,{}))

    def test_observed_markdown_and_native_xml_block_variants(self):
        for fmt,content in [('xml','<p id="'+native.TARGET+'">New <b><a href="synthetic://reference">status</a></b> — 你好</p>'),
                            ('xml','New <b><a href="synthetic://reference">status</a></b> — 你好'),
                            ('markdown','New **[status](synthetic://reference)** — 你好')]:
            with self.subTest(format=fmt,content=content):
                s=docs_edit.Scenario();r=[]
                self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'block_replace','block_id':native.TARGET,'format':fmt,'content':content})
                self.assertNotEqual(s.state['blocks'][1]['block_id'],native.TARGET)
                self.assertTrue(docs_edit.verify(None,native.EXPECTED_ANSWER,self.receipt(docs_edit,s,r),{}))

    def test_markdown_narrow_edit_rebuilds_block_and_cross_mark_pattern_fails(self):
        s=docs_edit.Scenario();r=[]
        data=self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'str_replace','format':'markdown','pattern':'Old status','content':'New status'})
        self.assertEqual(data['result'],'failed')
        self.assertEqual(s.state,native.INITIAL)
        self.assertFalse(docs_edit.verify(None,native.EXPECTED_ANSWER,self.receipt(docs_edit,s,r),{}))
        self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'str_replace','format':'markdown','pattern':'Old','content':'New'})
        self.assertNotEqual(s.state['blocks'][1]['block_id'],native.TARGET)
        self.assertTrue(docs_edit.verify(None,native.EXPECTED_ANSWER,self.receipt(docs_edit,s,r),{}))

    def test_unobserved_markdown_and_xml_remain_unsupported_before_mutation(self):
        for command,fmt,content in [('block_replace','markdown','New [**status**](synthetic://reference) — 你好'),
            ('block_replace','markdown','New _status_ — 你好'),
            ('block_replace','markdown','New **[`status`](synthetic://reference)** — 你好'),('block_replace','markdown','First\n\nSecond'),
            ('block_replace','xml','<p unknown="1">New status</p>'),('str_replace','markdown','New **[status](synthetic://reference)**')]:
            s=docs_edit.Scenario();r=[]
            body={'command':command,'format':fmt,'content':content}
            body.update({'block_id':native.TARGET} if command=='block_replace' else {'pattern':'Old status'})
            with self.subTest(body=body),self.assertRaisesRegex(ValueError,'^unsupported strategy:'):
                self.call(docs_edit,s,r,'PUT',native.AI_PATH,body)
            self.assertEqual(s.state,native.INITIAL)
            self.assertEqual(s.mutations,[])

    def test_markdown_duplicate_pattern_in_one_paragraph_rejects(self):
        s=docs_edit.Scenario();r=[]
        target=s.state['blocks'][1]
        target['runs'][0]['text']='Old Old '
        target['text']='Old Old status — 你好'
        before=copy.deepcopy(s.state)
        with self.assertRaisesRegex(ValueError,'ambiguous literal match'):
            self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'str_replace','format':'markdown','pattern':'Old','content':'New'})
        self.assertEqual(s.state,before)
        self.assertEqual(s.mutations,[])

    def test_failure_diagnostics_distinguish_content_state_and_proof(self):
        s=docs_edit.Scenario();r=[]
        self.call(docs_edit,s,r,'PUT',native.AI_PATH,{'command':'str_replace','format':'xml','pattern':'Old','content':'New'})
        good=self.receipt(docs_edit,s,r)
        diagnose=lambda receipt,answer=native.EXPECTED_ANSWER:docs_edit.verification_diagnostics(None,answer,receipt,{})
        self.assertEqual(diagnose(good),[])
        self.assertEqual(diagnose(good,{}),['paired_docs_answer_mismatch'])
        bad=copy.deepcopy(good);bad['state']['final']['blocks'][1]['runs'][1]['style']['bold']=False
        self.assertEqual(diagnose(bad),['paired_docs_bold_mismatch'])
        bad=copy.deepcopy(good);bad['state']['final']['blocks'][1]['runs'][1]['style']['link']['url']='private secret'
        self.assertEqual(diagnose(bad),['paired_docs_link_mismatch'])
        bad=copy.deepcopy(good);bad['state']['final']['blocks'][2]['text']='private secret'
        self.assertEqual(diagnose(bad),['paired_docs_unrelated_state_changed'])
        bad=copy.deepcopy(good);bad['state']['final']['blocks'][1]['runs'][0]['text']='Wrong '
        bad['state']['final']['blocks'][1]['text']='Wrong status — 你好'
        self.assertEqual(diagnose(bad),['paired_docs_text_mismatch'])
        bad=copy.deepcopy(good);bad['requests']=[]
        self.assertEqual(diagnose(bad),['paired_docs_proof_mismatch'])
        self.assertFalse(docs_edit.verify(None,native.EXPECTED_ANSWER,bad,{}))
        self.assertNotIn('private',json.dumps(diagnose(bad)))

    def test_old_module_plain_semantics_unchanged_and_public_schema_no_answers(self):
        s=native.Scenario();s.dispatch('PUT',native.AI_PATH,json.dumps({'command':'str_replace','pattern':'Old status','content':'New status'}).encode(),{'X-Lark-Proxy-Identity':'user'})
        self.assertTrue(s.state['blocks'][1]['runs'][1]['style']['bold'])
        for module in (docs_read,docs_edit):
            schema=json.dumps(module.TASK['fixture']['answer_schema'])
            self.assertNotIn('const',schema);self.assertNotIn('enum',schema);self.assertNotIn(docs_read.FACT,schema)


@unittest.skipUnless(os.environ.get('EVAL_PAIRED_BINARIES'),'requires two clean archived binaries')
class DocsPairedBinaryTests(unittest.TestCase):
    def test_both_binaries_read_and_valid_edit_strategies(self):
        from scenario_fixture import ScenarioFixture
        from codex_runner import cli_environment
        for binary in json.loads(os.environ['EVAL_PAIRED_BINARIES']):
            for kind in ('read_fetch','read_blocks','edit_xml','edit_narrow','edit_patch','edit_block','edit_block_id','edit_block_inline','edit_md_block','edit_md_narrow'):
                module=docs_read if kind.startswith('read') else docs_edit;s=module.Scenario()
                with self.subTest(binary=Path(binary).name,strategy=kind),tempfile.TemporaryDirectory() as directory,ScenarioFixture(module.TASK['fixture']['scenario'],s) as server:
                    work=Path(directory)
                    for name in ('home','tmp','feishu-config'):(work/name).mkdir()
                    env={**cli_environment(work),**server.environment()}
                    if kind=='read_fetch':args=['docs','+fetch','--doc',native.DOCUMENT,'--detail','full']
                    elif kind=='read_blocks':args=['api','GET',native.DOCX_PATH+'/blocks']
                    elif kind=='edit_patch':
                        els=native.wire_block(s.state['blocks'][1])['text']['elements'];els[0]['text_run']['content']='New '
                        args=['api','PATCH',native.DOCX_PATH+'/blocks/'+native.TARGET,'--data',json.dumps({'update_text_elements':{'elements':els}})]
                    elif kind in ('edit_block','edit_block_id','edit_block_inline','edit_md_block'):
                        content='<p>New <a href="synthetic://reference"><b>status</b></a> — 你好</p>'
                        if kind=='edit_block_id':content=content.replace('<p>','<p id="'+native.TARGET+'">')
                        if kind=='edit_block_inline':content=content[3:-4]
                        if kind=='edit_md_block':content='New **[status](synthetic://reference)** — 你好'
                        args=['docs','+update','--doc',native.DOCUMENT,'--command','block_replace','--block-id',native.TARGET,'--content',content,'--doc-format','markdown' if kind=='edit_md_block' else 'xml']
                    elif kind=='edit_md_narrow':args=['docs','+update','--doc',native.DOCUMENT,'--command','str_replace','--pattern','Old','--content','New','--doc-format','markdown']
                    else:args=['docs','+update','--doc',native.DOCUMENT,'--command','str_replace','--pattern','Old' if kind=='edit_narrow' else 'Old status','--content','New' if kind=='edit_narrow' else 'New <a href="synthetic://reference"><b>status</b></a>']
                    result=subprocess.run([binary,*args,'--as','user'],cwd=work,env=env,text=True,capture_output=True,timeout=30)
                    self.assertEqual(result.returncode,0,result.stderr)
                    answer=docs_read.ANSWER if kind.startswith('read') else native.EXPECTED_ANSWER
                    self.assertTrue(module.verify(work,answer,server.receipt(),module.TASK['expected']))
