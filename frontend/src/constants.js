import { t } from './i18n/index.js'
export const APP_NAME = 'WinToolBox' // synced from version.json
export const APP_VERSION = 'v1.1.1' // synced from version.json

export const MENUS = [
  {
    key: 'overview',
    get title() {
      return t('系统信息')
    },
    icon: 'Monitor',
    get desc() {
      return t('系统与硬件')
    },
  },
  {
    key: 'account',
    get title() {
      return t('账户管理')
    },
    icon: 'User',
    get desc() {
      return t('用户、密码与登录安全')
    },
  },
  {
    key: 'rdp',
    get title() {
      return t('远程桌面')
    },
    icon: 'Connection',
    get desc() {
      return t('连接设置与访问限制')
    },
  },
  {
    key: 'firewall',
    get title() {
      return t('防火墙')
    },
    icon: 'Lock',
    get desc() {
      return t('一键开关、禁 ping 与高危端口')
    },
  },
  {
    key: 'harden',
    get title() {
      return t('安全设置')
    },
    icon: 'Key',
    get desc() {
      return t('SMBv1、WinRM 与匿名限制')
    },
  },
  {
    key: 'defender',
    get title() {
      return t('防病毒')
    },
    icon: 'FirstAidKit',
    get desc() {
      return t('Defender 实时防护')
    },
  },
  {
    key: 'time',
    get title() {
      return t('日期和时间')
    },
    icon: 'Clock',
    get desc() {
      return t('时区与 NTP')
    },
  },
  {
    key: 'power',
    get title() {
      return t('关机重启')
    },
    icon: 'SwitchButton',
    get desc() {
      return t('锁定 / 重启 / 关机')
    },
  },
  {
    key: 'systemlogs',
    get title() {
      return t('系统日志')
    },
    icon: 'Clock',
    get desc() {
      return t('查看重启原因和系统异常')
    },
  },
  {
    key: 'update',
    get title() {
      return t('Windows 更新')
    },
    icon: 'Refresh',
    get desc() {
      return t('Windows Update 开关')
    },
  },
  {
    key: 'selfupdate',
    get title() {
      return t('软件更新')
    },
    icon: 'Download',
    get desc() {
      return t('检查新版本并安装')
    },
  },
]

export const NTP_PRESETS = [
  { id: 'time.windows.com', label: 'Microsoft' },
  {
    id: 'ntp.aliyun.com',
    get label() {
      return t('阿里云')
    },
  },
  {
    id: 'ntp.tencent.com',
    get label() {
      return t('腾讯云')
    },
  },
  { id: 'cn.pool.ntp.org', label: 'NTP Pool' },
  {
    id: 'ntp.ntsc.ac.cn',
    get label() {
      return t('国家授时')
    },
  },
  { id: 'time.nist.gov', label: 'NIST' },
]

// Labels wrap rather than truncate when English text is longer than Chinese.
export const FORM_LABEL_WIDTH = '150px'
export const FORM_LABEL_WIDTH_WIDE = '180px'
