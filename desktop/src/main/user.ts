import { execFile } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { userInfo } from 'node:os';
import type { UserInfo } from '../shared/api';

function run(cmd: string, args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile(cmd, args, { timeout: 2000 }, (err, stdout) => (err ? reject(err) : resolve(stdout.trim())));
  });
}

/** The GECOS name of a login in /etc/passwd, the field Linux keeps a full name in. */
async function gecosName(login: string): Promise<string> {
  const passwd = await readFile('/etc/passwd', 'utf8');
  const line = passwd.split('\n').find((l) => l.startsWith(`${login}:`));
  return (line?.split(':')[4] ?? '').split(',')[0]?.trim() ?? '';
}

/**
 * Who is using this machine, as the sidebar footer shows them: the full name
 * the system has, else the login. The login is always there; a lookup that
 * fails is written to the log and the login is used, never an invented name.
 */
export async function currentUser(): Promise<UserInfo> {
  const login = userInfo().username;
  try {
    const full =
      process.platform === 'darwin' ? await run('id', ['-F']) : process.platform === 'linux' ? await gecosName(login) : '';
    return { name: full || login };
  } catch (err) {
    console.warn(`dcode: could not read the full name of ${login}, showing the login instead:`, err);
    return { name: login };
  }
}
