// Run against the dedicated local prototype only, after sourcing .envrc.
// Creates four ordinary users and retains a test post/image for replacement checks.
import assert from 'node:assert/strict';
import { mkdir, readFile, writeFile } from 'node:fs/promises';

const origin = 'http://localhost:3005';
const statePath = new URL('../data/prototype-smoke.json', import.meta.url);
const verify = process.argv.includes('--verify');
const image = await readFile(new URL('../../testdata/images/test.png', import.meta.url));

async function request(path, {cookie, json, body, method = 'GET', status = 200} = {}) {
  const headers = {};
  if (cookie) headers.Cookie = cookie;
  if (json) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(json); }
  const res = await fetch(origin + path, {method, headers, body, redirect: 'manual', signal: AbortSignal.timeout(30000)});
  assert.equal(res.status, status, `${method} ${path}: unexpected status`);
  return res;
}

async function login(name, password) {
  assert.ok(password, `Missing password for ${name}`);
  const res = await request('/api/v3/login', {method:'POST', json:{name,password}, status:204});
  const cookie = res.headers.getSetCookie().map(v => v.split(';')[0]).join('; ');
  assert.ok(cookie, 'Login must set a session cookie');
  return cookie;
}

async function fileBytes(id, cookie) {
  const res = await request(`/api/v3/files/${id}`, {cookie, status:302});
  // Do not forward the traQ cookie to S3 or print signed URLs.
  const remote = await fetch(res.headers.get('location'), {signal:AbortSignal.timeout(30000)});
  assert.equal(remote.status, 200);
  return Buffer.from(await remote.arrayBuffer());
}

await request('/api/ping');
await request('/api/v3/users', {method:'POST',json:{name:'signup_probe',password:'not-used-password'},status:401});
await request('/api/v3/login', {method:'POST', json:{name:'traq',password:'traq'},status:401});
const admin = await login('traq', process.env.TRAQ_INITIALADMINPASSWORD);
const users = await (await request('/api/v3/users', {cookie:admin})).json();
const sessions = [];
for (let i=1; i<=4; i++) {
  const name = `demo0${i}`;
  const password = process.env[`PROTOTYPE_DEMO0${i}_PASSWORD`];
  assert.ok(password, `Missing password for ${name}`);
  if (!verify && !users.some(u => u.name === name)) {
    await request('/api/v3/users', {cookie:admin,method:'POST',json:{name,password},status:201});
  }
  const cookie = await login(name,password);
  const me = await (await request('/api/v3/users/me', {cookie})).json();
  assert.ok(!me.permissions.includes('register_user'), 'Demo users must not administer users');
  sessions.push({cookie,me});
}
console.log('Health, custom admin password, and four ordinary demo logins verified.');

const {cookie,me} = sessions[0];
let state;
if (verify) {
  state = JSON.parse(await readFile(statePath,'utf8'));
} else {
  // Replace only the previous run's own artifacts, never unrelated posts/files.
  let previous;
  try { previous = JSON.parse(await readFile(statePath,'utf8')); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (previous) {
    await request(`/api/v3/messages/${previous.messageId}`, {cookie,method:'DELETE',status:204});
    await request(`/api/v3/files/${previous.fileId}`, {cookie,method:'DELETE',status:204});
  }
  const channels = await (await request('/api/v3/channels', {cookie})).json();
  const channel = channels.public.find(ch => ch.name === 'general');
  assert.ok(channel);
  const form = new FormData();
  form.append('channelId',channel.id);
  form.append('file',new Blob([image], {type:'image/png'}),'prototype-smoke.png');
  const file = await (await request('/api/v3/files', {cookie,method:'POST',body:form,status:201})).json();
  const content = `Prototype deployment smoke check: ${new Date().toISOString()}`;
  const message = await (await request(`/api/v3/channels/${channel.id}/messages`, {cookie,method:'POST',json:{content},status:201})).json();
  state = {fileId:file.id,messageId:message.id,content};
  await mkdir(new URL('../data/',import.meta.url), {recursive:true});
  await writeFile(statePath,JSON.stringify(state,null,2)+'\n', {mode:0o600});
}
const message = await (await request(`/api/v3/messages/${state.messageId}`, {cookie:sessions[1].cookie})).json();
assert.equal(message.content,state.content);
assert.deepEqual(await fileBytes(state.fileId,cookie),image);
const thumbnail = await request(`/api/v3/files/${state.fileId}/thumbnail`, {cookie});
assert.ok((await thumbnail.arrayBuffer()).byteLength > 0);
const icon = await request(`/api/v3/users/${me.id}/icon`, {cookie});
assert.ok((await icon.arrayBuffer()).byteLength > 0);
console.log(`${verify ? 'After replacement: ' : ''}cross-user post read, S3 image, thumbnail, and icon verified.`);
