// The whole surface the preload hands the renderer. Narrow on purpose: the
// renderer never sees Node, the filesystem or the daemon's socket, only these.

export interface UserInfo {
  /** The person's name as the system knows it, or their login when it has none. */
  name: string;
}

export interface DcodeApi {
  /** process.platform of the main process: decides where the native window controls are. */
  readonly platform: string;
  user(): Promise<UserInfo>;
}

/** The channel the main process answers `user()` on. */
export const USER_CHANNEL = 'dcode:user';
