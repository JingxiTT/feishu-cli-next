"""Independent paired Markdown state and failure controls; no models/live APIs."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
sys.path.insert(0,str(Path(__file__).resolve().parents[2]/'evaluation'))
from paired import markdown as m


class MarkdownPairedTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.work=Path(self.temp.name);self.s=m.Scenario();self.s.setup(self.work);self.records=[]
    def call(self,method,path,data):
        reply=self.s.dispatch(method,path,json.dumps(data).encode(),{'X-Lark-Proxy-Identity':'user'})
        self.records.append({'route':reply.route,'status':reply.status,'proof':copy.deepcopy(reply.proof)})
        return reply.json['data']
    def receipt(self):return {'scenario':m.TASK['fixture']['scenario'],'state':self.s.snapshot(),'requests':self.records}
    def valid(self):return m.verify(self.work,{'document_id':m.DOCUMENT},self.receipt(),{})
    def test_markdown_creates_native_actual_state_and_equivalent_xml(self):
        for fmt,content in [('markdown',m.SOURCE),('xml',m.xml_content(m.EXPECTED_NODES,False))]:
            with self.subTest(format=fmt):
                self.s=m.Scenario();self.records=[]
                self.call('POST',m.CREATE,{'format':fmt,'content':content})
                self.assertTrue(self.valid())
                response=self.call('POST',m.FETCH,{'format':'xml','export_option':{'export_block_id':True}})
                self.assertIn('<table',response['document']['content'])
                self.assertTrue(self.valid())
    def test_wrong_content_applies_and_fails_not_repaired(self):
        for change in [m.SOURCE.replace('**bold**','bold'),m.SOURCE.replace('你好','lost'),m.SOURCE.replace('001 & <literal>','bad')]:
            self.s=m.Scenario();self.records=[]
            self.call('POST',m.CREATE,{'format':'markdown','content':change});self.assertFalse(self.valid())
    def test_forged_state_answer_no_request_and_changed_source_fail(self):
        self.assertFalse(self.valid())
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE});self.assertTrue(self.valid())
        receipt=self.receipt();receipt['requests']=[]
        self.assertFalse(m.verify(self.work,{'document_id':m.DOCUMENT},receipt,{}))
        receipt=self.receipt();receipt['state']['document']['nodes'][0]['text']='forged'
        self.assertFalse(m.verify(self.work,{'document_id':m.DOCUMENT},receipt,{}))
        self.assertFalse(m.verify(self.work,{'document_id':'wrong'},self.receipt(),{}))
        (self.work/'input.md').write_text('changed');self.assertFalse(self.valid())
    def test_failure_diagnostics_are_specific_without_content(self):
        reasons=m.verification_diagnostics(self.work,{'document_id':m.DOCUMENT},self.receipt(),{})
        self.assertIn('paired_markdown_document_missing',reasons)
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        self.assertEqual(m.verification_diagnostics(self.work,{'document_id':m.DOCUMENT},self.receipt(),{}),[])
        self.s.document['nodes'][1]['characters']=[]
        reasons=m.verification_diagnostics(self.work,{'document_id':m.DOCUMENT},self.receipt(),{})
        self.assertEqual(reasons,['paired_markdown_native_content_mismatch'])
        self.assertNotIn('Keep',json.dumps(reasons))
        self.assertFalse(self.valid())

    def test_revision_selectors_are_checked(self):
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        for value in [-1,1]:
            self.call('POST',m.FETCH,{'revision_id':value})
            self.call('GET',m.DOCX+'/blocks?document_revision_id='+str(value),{})
        for value in [0,2,True,'bad']:
            with self.assertRaises(ValueError):self.call('POST',m.FETCH,{'revision_id':value})
            with self.assertRaises(ValueError):self.call('GET',m.DOCX+'/blocks?document_revision_id='+str(value),{})
        self.assertTrue(self.valid())
    def test_unsupported_is_not_unauthorized_or_invalid(self):
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        for method,path,body in [('DELETE',m.DOCX,{}),('GET',m.DOCX+'foreign',{}),('POST',m.FETCH,{'unknown':True})]:
            with self.assertRaises(ValueError) as caught:self.call(method,path,body)
            self.assertNotIn('unsupported strategy:',str(caught.exception))
        self.assertIn('**bold**',self.call('POST',m.FETCH,{'format':'markdown'})['document']['content'])
        with self.assertRaisesRegex(ValueError,'^unsupported strategy:'):
            self.call('POST',m.FETCH,{'read_option':{'read_mode':'keyword','keyword':'bold','context_after':1}})
        self.assertTrue(self.valid())

    def test_native_markdown_and_scoped_readbacks_do_not_change_state(self):
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        result=self.call('POST',m.FETCH,{'format':'markdown'})['document']['content']
        self.assertEqual(result,m.SOURCE.strip().replace('| --- | --- |','|-|-|'))
        for mode,options in [('outline',{}),('keyword',{'keyword':'bold'}),
            ('range',{'start_block_id':'pairedBlock0','end_block_id':'pairedBlock0'}),
            ('section',{'start_block_id':'pairedBlock0'})]:
            result=self.call('POST',m.FETCH,{'export_option':{'export_block_id':True},
                'read_option':{'read_mode':mode,**options}})['document']['content']
            self.assertIn('<fragment mode="'+mode+'"',result)
            self.assertNotIn('block-id=',result)
            self.assertEqual('pairedBlock1' in result,mode in ('keyword','section'))
            self.assertEqual('<table' in result,mode=='section')
        self.assertTrue(self.valid())
        self.s.document['nodes'][1]['characters']=[]
        result=self.call('POST',m.FETCH,{'format':'markdown'})['document']['content']
        self.assertNotIn('**bold**',result)
        self.assertFalse(self.valid())

    def test_xml_language_and_unexpected_table_text(self):
        xml=m.xml_content(m.EXPECTED_NODES,False)
        self.assertIn('lang="text"',xml)
        self.assertEqual(m.parse_xml(xml)[3],m.EXPECTED_NODES[3])
        self.assertEqual(m.parse_xml(xml.replace('lang="text"','lang="Go"'))[3]['language'],'Go')
        for before,after in [('<table>','<table>lost'),('</tr>','</tr>lost'),('</td>','</td>lost'),('</p></td>','</p>lost</td>')]:
            with self.assertRaises(ValueError):m.parse_xml(xml.replace(before,after,1))
        with self.assertRaisesRegex(ValueError,'^unsupported strategy:'):
            m.parse_markdown('Keep *italic*.')
        with self.assertRaisesRegex(ValueError,'^unsupported strategy:'):
            m.parse_xml('<blockquote><p>valid but unmodeled</p></blockquote>')
        with self.assertRaisesRegex(ValueError,'^invalid XML conversion'):
            m.parse_xml('<p>unclosed')

    def test_duplicate_create_foreign_parent_invalid_json_and_xml_reject(self):
        for content in ['<!DOCTYPE x><p>bad</p>','<p><em>unsupported</em></p>']:
            with self.assertRaises(ValueError):self.call('POST',m.CREATE,{'format':'xml','content':content})
        with self.assertRaises(ValueError):self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE,'parent_token':'foreign'})
        self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        with self.assertRaises(ValueError):self.call('POST',m.CREATE,{'format':'markdown','content':m.SOURCE})
        self.assertTrue(self.valid())


@unittest.skipUnless(os.environ.get('EVAL_PAIRED_BINARIES'),'requires two clean archived binaries')
class MarkdownPairedBinaryTests(unittest.TestCase):
    def test_both_binaries_native_create_and_import(self):
        from scenario_fixture import ScenarioFixture
        from codex_runner import cli_environment
        for binary in json.loads(os.environ['EVAL_PAIRED_BINARIES']):
            for strategy in ('markdown','xml','import'):
                s=m.Scenario()
                with self.subTest(binary=Path(binary).name,strategy=strategy),tempfile.TemporaryDirectory() as directory,ScenarioFixture(m.TASK['fixture']['scenario'],s) as server:
                    work=Path(directory);s.setup(work)
                    for name in ('home','tmp','feishu-config'):(work/name).mkdir()
                    env={**cli_environment(work),**server.environment()}
                    if strategy=='import':args=['drive','+import','--file','input.md','--type','docx']
                    else:args=['docs','+create','--doc-format',strategy,'--content','@input.md' if strategy=='markdown' else m.xml_content(m.EXPECTED_NODES,False)]
                    result=subprocess.run([binary,*args,'--as','user'],cwd=work,env=env,text=True,capture_output=True,timeout=30)
                    self.assertEqual(result.returncode,0,result.stderr)
                    result=subprocess.run([binary,'docs','+fetch','--doc',m.DOCUMENT,'--detail','full','--as','user'],cwd=work,env=env,text=True,capture_output=True,timeout=30)
                    self.assertEqual(result.returncode,0,result.stderr)
                    for extra in (['--doc-format','markdown'],['--scope','outline'],
                                  ['--scope','keyword','--keyword','bold'],
                                  ['--scope','section','--start-block-id','pairedBlock0'],
                                  ['--scope','range','--start-block-id','pairedBlock0','--end-block-id','pairedBlock0']):
                        result=subprocess.run([binary,'docs','+fetch','--doc',m.DOCUMENT,'--detail','full',*extra,'--as','user'],cwd=work,env=env,text=True,capture_output=True,timeout=30)
                        self.assertEqual(result.returncode,0,result.stderr)
                    self.assertTrue(m.verify(work,{'document_id':m.DOCUMENT},server.receipt(),m.TASK['expected']))
