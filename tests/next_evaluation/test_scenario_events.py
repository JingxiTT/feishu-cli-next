"""Offline event fixture/verifier tests; optional clean-archive CLI subprocesses."""
import copy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shlex
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scenarios.events import APP_ID, EXPECTED, KEY, NAME, PAYLOADS, Scenario, TASK, verify, verification_diagnostics, _executable
from section_fixture import valid_signature, PROXY_KEY

UNIX = sys.platform == "darwin" or sys.platform.startswith("linux")
CLIENT = r'''
import json,os,pathlib,socket,sys,time
conn=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM)
conn.connect(sys.argv[1])
reader=conn.makefile('rb')
hello={'type':'hello','pid':os.getpid(),'event_key':'im.chat.updated_v1','event_types':['im.chat.updated_v1'],'version':'v1'}
if sys.argv[3]=='foreign': hello['pid']+=100000
conn.sendall((json.dumps(hello)+'\n').encode())
if sys.argv[3]=='foreign':
 reader.readline();sys.exit(0)
assert json.loads(reader.readline())['type']=='hello_ack'
values=[json.loads(reader.readline())['payload'] for _ in range(2)]
pathlib.Path(sys.argv[2]).write_text(''.join(json.dumps(v,ensure_ascii=False)+'\n' for v in values),encoding='utf-8')
if sys.argv[3]=='leak': time.sleep(30)
conn.sendall(b'{"type":"bye"}\n')
reader.close();conn.close()
'''


def command(binary):
    return f"{shlex.quote(str(binary))} event consume {KEY} --as bot --max-events 2 --timeout 5s > events.ndjson"


class RouteAndVerifierTests(unittest.TestCase):
    def test_public_fixture_contains_only_types_and_scenario(self):
        self.assertEqual(set(TASK['fixture']), {'scenario', 'answer_schema'})
        public = json.dumps(TASK['fixture'])
        for hidden in ['你好', 'e1', 'e2', 'oc_fixture_events', 'payloads']:
            self.assertNotIn(hidden, public)

    def test_routes_validate_method_identity_target_and_query(self):
        scenario=Scenario()
        bot={'X-Lark-Proxy-Identity':'bot'}
        route=f'/open-apis/application/v6/applications/{APP_ID}/app_versions'
        for query in ['', '?lang=zh_cn&page_size=2', '?lang=en_us&page_size=1']:
            reply=scenario.dispatch('GET',route+query,b'',bot)
            self.assertEqual(reply.route,'event_preflight')
            self.assertEqual(reply.json,{'code':0,'data':{'items':[]}})
        for method,uri,body,headers in [
            ('POST',route,b'{}',bot),('GET',route,b'{}',bot),
            ('GET',route,b'',{'X-Lark-Proxy-Identity':'user'}),('GET',route,b'',{}),
            ('GET',route+'?page_size=0',b'',bot),('GET',route+'?page_size=2&page_size=3',b'',bot),
            ('GET',route+'?page_token=foreign',b'',bot),
            ('GET',route.replace(APP_ID,'foreign-app'),b'',bot),
            ('GET','https://external.invalid'+route,b'',bot),
            ('DELETE','/open-apis/im/v1/messages/id',b'',bot),
            ('GET','/open-apis/event/v1/connection?foreign=true',b'',bot),
        ]:
            with self.subTest(uri=uri,method=method),self.assertRaises(ValueError):
                scenario.dispatch(method,uri,body,headers)
        self.assertEqual(scenario.dispatch('GET','/open-apis/authen/v1/user_info',b'',bot).route,'identity')

    def test_execution_proof_does_not_treat_complex_shell_as_child_exit(self):
        scenario=Scenario();scenario.bind_cli(sys.executable)
        binary=scenario.selected_cli
        good=command(binary)
        for cmd in [good, '/bin/zsh -lc '+shlex.quote(good)]:
            scenario.observe_execution({'type':'command_execution','command':cmd,'exit_code':0})
        self.assertEqual(len(scenario.snapshot()['executions']),2)
        for cmd in [good+' || true',good+'; exit 0',good+' | cat',good+' & wait',
                    'echo '+good,'$CLI event consume '+KEY,good+'$(false)']:
            scenario.observe_execution({'type':'command_execution','command':cmd,'exit_code':0})
        self.assertEqual(len(scenario.snapshot()['executions']),2)
        scenario.observe_execution({'type':'command_execution','command':good,'exit_code':False})
        self.assertEqual(len(scenario.snapshot()['executions']),2)

    def test_provider_completed_commands_resolve_only_trusted_cli_aliases(self):
        with tempfile.TemporaryDirectory() as directory:
            scenario=Scenario();scenario.bind_cli(sys.executable)
            scenario.workspace=Path(directory)
            bindir=scenario.workspace/'bin';bindir.mkdir()
            for alias in ('lark-cli','feishu-cli-next','arbitrary-cli'):
                try:
                    (bindir/alias).symlink_to(scenario.selected_cli)
                except OSError as error:
                    self.skipTest(f'symlink aliases unavailable: {error}')
            for alias in ('lark-cli','feishu-cli-next'):
                event={'type':'item.completed','item':{'id':'controlled-command','type':'command_execution',
                    'command':'/bin/zsh -lc '+shlex.quote(command(alias)),
                    'aggregated_output':'','exit_code':0,'status':'completed'}}
                scenario.observe_execution(event['item'])
            self.assertEqual(scenario.snapshot()['executions'],[
                {'simple_selected_cli_consume':True,'exit_code':0},
                {'simple_selected_cli_consume':True,'exit_code':0}])
            scenario.observe_execution({'type':'command_execution','command':command('arbitrary-cli'),'exit_code':0})
            self.assertEqual(len(scenario.snapshot()['executions']),2,'arbitrary bare names are not trusted PATH aliases')
            (bindir/'lark-cli').unlink()
            (bindir/'lark-cli').write_text('not the selected executable')
            scenario.observe_execution({'type':'command_execution','command':command('lark-cli'),'exit_code':0})
            (bindir/'feishu-cli-next').unlink()
            scenario.observe_execution({'type':'command_execution','command':command('feishu-cli-next'),'exit_code':0})
            self.assertEqual(len(scenario.snapshot()['executions']),2,'missing/foreign aliases must not create proof')

    def test_missing_or_malformed_independent_evidence_fails(self):
        for receipt in [None, [], {}, {'scenario':NAME}, {'scenario':NAME,'requests':[None]}]:
            self.assertFalse(verify('/not-used',EXPECTED['answer'],receipt,EXPECTED))
            self.assertEqual(verification_diagnostics('/not-used',EXPECTED['answer'],receipt,EXPECTED),[])

    def test_missing_exit_diagnostic_requires_every_other_predicate(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact=Path(directory)/'events.ndjson'
            artifact.write_text(''.join(json.dumps(p,ensure_ascii=False)+'\n' for p in PAYLOADS),encoding='utf-8')
            receipt={'scenario':NAME,'requests':[{'route':'event_preflight','status':200}],
                     'state':{'scenario':NAME,'rejected':[],'active_before_teardown':0,
                              'selected_cli':'selected-cli','executions':[],
                              'episodes':[{'delivered_ids':['e1','e2'],'connection_closed':True,
                                           'terminated':True,'forced_disconnect':False,
                                           'executable':'selected-cli','peer_pid':123,'termination_seconds':0.1}]}}
            def diagnostics(value=receipt,answer=EXPECTED['answer']):
                self.assertFalse(verify(directory,answer,value,EXPECTED))
                return verification_diagnostics(directory,answer,value,EXPECTED)
            self.assertEqual(diagnostics(),['execution_proof_unavailable'])
            partial=copy.deepcopy(receipt)
            partial['state']['episodes'].append(copy.deepcopy(partial['state']['episodes'][0]))
            partial['state']['executions']=[{'simple_selected_cli_consume':True,'exit_code':0}]
            self.assertEqual(diagnostics(partial),['execution_proof_unavailable'])
            for change in ['nonzero','malformed_exit','boolean_exit','extra_exit','wrong_peer','leak','forced',
                           'missing_event','unclosed','rejected','no_preflight','bad_api','bad_duration']:
                with self.subTest(change=change):
                    damaged=copy.deepcopy(receipt);state=damaged['state'];episode=state['episodes'][0]
                    if change=='nonzero':
                        state['episodes'].append(copy.deepcopy(episode))
                        state['executions']=[{'simple_selected_cli_consume':True,'exit_code':1}]
                    if change=='malformed_exit':state['executions']=[None]
                    if change=='boolean_exit':state['executions']=[{'simple_selected_cli_consume':True,'exit_code':False}]
                    if change=='extra_exit':state['executions']=[{'simple_selected_cli_consume':True,'exit_code':0}]*2
                    if change=='wrong_peer':episode['executable']='foreign-cli'
                    if change=='leak':state['active_before_teardown']=1;episode['terminated']=False
                    if change=='forced':episode['forced_disconnect']=True
                    if change=='missing_event':episode['delivered_ids']=['e1']
                    if change=='unclosed':episode['connection_closed']=False
                    if change=='rejected':state['rejected']=['invalid_or_interrupted_connection']
                    if change=='no_preflight':damaged['requests']=[]
                    if change=='bad_api':damaged['requests'][0]['status']=403
                    if change=='bad_duration':episode['termination_seconds']=True
                    self.assertEqual(diagnostics(damaged),[])
            self.assertEqual(diagnostics(answer={'events':[]}),[])
            artifact.write_text('{}\n',encoding='utf-8')
            self.assertEqual(diagnostics(),[])


@unittest.skipUnless(UNIX,'event IPC peer identity fixture is macOS/Linux only')
class UnixScenarioTests(unittest.TestCase):
    def setup_episode(self, directory, mode='normal'):
        scenario=Scenario();scenario.bind_cli(_executable(os.getpid()));scenario.setup(directory)
        self.addCleanup(scenario.close)
        process=subprocess.Popen([sys.executable,'-c',CLIENT,scenario.unix_sockets()[0],
                                  str(Path(directory)/'events.ndjson'),mode],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        return scenario,process

    def receipt(self, scenario):
        return {'scenario':NAME,'requests':[{'route':'event_preflight','status':200,
                'proof':{'app_id':APP_ID}}],'state':scenario.snapshot()}

    def test_real_socket_delivery_peer_identity_exit_and_tamper_rejections(self):
        with tempfile.TemporaryDirectory() as directory:
            scenario,process=self.setup_episode(directory)
            _,err=process.communicate(timeout=10)
            self.assertEqual(process.returncode,0,err)
            scenario.observe_execution({'type':'command_execution','command':command(scenario.selected_cli),'exit_code':0})
            scenario.close();scenario.close()
            receipt=self.receipt(scenario)
            self.assertEqual(receipt['state']['episodes'][0]['peer_pid'],process.pid)
            self.assertTrue(verify(directory,EXPECTED['answer'],receipt,EXPECTED),receipt)
            self.assertEqual(verification_diagnostics(directory,EXPECTED['answer'],receipt,EXPECTED),[])
            artifact=Path(directory)/'events.ndjson';original=artifact.read_bytes()
            for value in ['', '{}\n', original.decode().replace('你好','other'),
                          original.decode().replace('"name": ""','"name": null'),
                          original.decode()+original.decode().splitlines()[0]+'\n']:
                artifact.write_text(value)
                self.assertFalse(verify(directory,EXPECTED['answer'],receipt,EXPECTED))
            artifact.write_bytes(original)
            for change in ['no_episode','no_execution','nonzero','foreign_executable','not_terminated','boolean_active','boolean_duration','mutated']:
                damaged=copy.deepcopy(receipt)
                if change=='no_episode':damaged['state']['episodes']=[]
                if change=='no_execution':damaged['state']['executions']=[]
                if change=='nonzero':damaged['state']['executions'][0]['exit_code']=1
                if change=='foreign_executable':damaged['state']['episodes'][0]['executable']='/foreign-cli'
                if change=='not_terminated':damaged['state']['episodes'][0]['terminated']=False
                if change=='boolean_active':damaged['state']['active_before_teardown']=False
                if change=='boolean_duration':damaged['state']['episodes'][0]['termination_seconds']=True
                if change=='mutated':damaged['requests'].append({'route':'rejected','status':403})
                self.assertFalse(verify(directory,EXPECTED['answer'],damaged,EXPECTED),change)
            self.assertFalse(verify(directory,{'events':[{'id':'e1','text':'你好'},{'id':'e2','text':None}]},receipt,EXPECTED))
            self.assertFalse(Path(scenario.unix_sockets()[0]).exists())

    def test_spoofed_pid_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            scenario,process=self.setup_episode(directory,'foreign')
            process.communicate(timeout=5);scenario.close()
            self.assertTrue(scenario.snapshot()['rejected'])
            self.assertEqual(scenario.snapshot()['episodes'],[])

    def test_live_consumer_is_failure_before_fixture_teardown(self):
        with tempfile.TemporaryDirectory() as directory:
            scenario,process=self.setup_episode(directory,'leak')
            try:
                deadline=time.monotonic()+3
                while not (Path(directory)/'events.ndjson').exists() and time.monotonic()<deadline:
                    time.sleep(.01)
                scenario.close()
                state=scenario.snapshot()
                self.assertGreater(state['active_before_teardown'],0)
                self.assertFalse(verify(directory,EXPECTED['answer'],self.receipt(scenario),EXPECTED))
            finally:
                process.terminate();process.communicate(timeout=5)


@unittest.skipUnless(UNIX and os.environ.get('EVAL_EVENTS_BINARIES'),
                     'set EVAL_EVENTS_BINARIES to clean archived authsidecar CLI paths')
class CleanCLIProcessTests(unittest.TestCase):
    def test_selected_cli_two_native_events_and_exit(self):
        pairs=[(binary,alias,mode) for binary in json.loads(os.environ['EVAL_EVENTS_BINARIES'])
               for alias in ('lark-cli','feishu-cli-next') for mode in ('direct','script')]
        for binary,alias,mode in pairs:
            with self.subTest(binary=Path(binary).name,alias=alias,mode=mode),tempfile.TemporaryDirectory() as directory:
                scenario=Scenario();scenario.bind_cli(binary);scenario.setup(directory)
                bindir=Path(directory)/'bin';bindir.mkdir()
                (bindir/alias).symlink_to(scenario.selected_cli)
                requests=[];lock=threading.Lock()
                class Handler(BaseHTTPRequestHandler):
                    def log_message(self,*_):pass
                    def do_GET(self):
                        body=b''
                        try:
                            if not valid_signature(self.command,self.path,body,self.headers):
                                raise ValueError('invalid signature')
                            reply=scenario.dispatch(self.command,self.path,body,self.headers)
                            result,status,route=reply.json,reply.status,reply.route
                        except ValueError:
                            result,status,route={'code':403},403,'rejected'
                        with lock:requests.append({'route':route,'status':status})
                        raw=json.dumps(result).encode()
                        self.send_response(status);self.send_header('Content-Type','application/json')
                        self.send_header('Content-Length',str(len(raw)));self.end_headers();self.wfile.write(raw)
                server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
                thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
                try:
                    # Inherit executable locations only, never credentials/profiles/proxies.
                    env={k:os.environ[k] for k in ('PATH','SYSTEMROOT') if k in os.environ}
                    env['PATH']=str(bindir)+os.pathsep+env.get('PATH','')
                    env.update(scenario.environment())
                    env.update({'HOME':directory,'LARKSUITE_CLI_AUTH_PROXY':f'http://127.0.0.1:{server.server_port}',
                                'LARKSUITE_CLI_PROXY_KEY':PROXY_KEY,'LARKSUITE_CLI_REMOTE_META':'off',
                                'LARKSUITE_CLI_NO_UPDATE_NOTIFIER':'1','LARKSUITE_CLI_NO_SKILLS_NOTIFIER':'1'})
                    argv=[alias,'event','consume',KEY,'--as','bot','--max-events','2','--timeout','5s']
                    if mode=='script':
                        script="import subprocess,sys\np=subprocess.run(sys.argv[1:],capture_output=True,timeout=10)\nsys.stdout.buffer.write(p.stdout)\nsys.stderr.buffer.write(p.stderr)\nsys.exit(p.returncode)"
                        argv=[sys.executable,'-c',script,*argv]
                    result=subprocess.run(argv,
                                          cwd=directory,env=env,capture_output=True,timeout=10)
                    self.assertEqual(result.returncode,0,result.stderr.decode())
                    self.assertIn(b'reason: limit',result.stderr)
                    (Path(directory)/'events.ndjson').write_bytes(result.stdout)
                    event={'type':'item.completed','item':{'type':'command_execution',
                           'command':shlex.join(argv) if mode=='script' else '/bin/zsh -lc '+shlex.quote(command(alias)),
                           'exit_code':result.returncode,'aggregated_output':''}}
                    scenario.observe_execution(event['item'])
                    scenario.close()
                    receipt={'scenario':NAME,'requests':requests,'state':scenario.snapshot()}
                    if mode=='script':
                        self.assertFalse(verify(directory,EXPECTED['answer'],receipt,EXPECTED))
                        self.assertEqual(verification_diagnostics(directory,EXPECTED['answer'],receipt,EXPECTED),
                                         ['execution_proof_unavailable'],receipt)
                    else:
                        self.assertTrue(verify(directory,EXPECTED['answer'],receipt,EXPECTED),receipt)
                        self.assertEqual(verification_diagnostics(directory,EXPECTED['answer'],receipt,EXPECTED),[])
                finally:
                    scenario.close();server.shutdown();server.server_close();thread.join(2)


if __name__=='__main__':unittest.main()
