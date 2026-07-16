type AvatarUser = {
  email?: string | null;
  avatar_url?: string | null;
};

const qqEmailPattern = /^([0-9]{5,12})@qq\.com$/i;

export function resolveAvatarURL(user: AvatarUser | null | undefined): string | undefined {
  const customAvatar = user?.avatar_url?.trim();
  if (customAvatar) return customAvatar;

  const email = user?.email?.trim().toLowerCase() ?? "";
  const match = qqEmailPattern.exec(email);
  if (!match) return undefined;
  return `https://q.qlogo.cn/headimg_dl?dst_uin=${match[1]}&spec=640&img_type=jpg`;
}
