import { t } from './i18n/index.js'
export const APP_NAME = 'WinToolBox' // synced from version.json
export const APP_VERSION = 'v1.0.0' // synced from version.json

export const MENUS = [
  {
    key: 'overview',
    get title() {
      return t('本机概览')
    },
    icon: 'Monitor',
    get desc() {
      return t('系统与硬件')
    },
  },
  {
    key: 'account',
    get title() {
      return t('本地账户')
    },
    icon: 'User',
    get desc() {
      return t('密码、权限与锁定策略')
    },
  },
  {
    key: 'rdp',
    get title() {
      return t('远程桌面')
    },
    icon: 'Connection',
    get desc() {
      return t('开关、端口、NLA 与连接记录')
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
      return t('安全加固')
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
      return t('时间同步')
    },
    icon: 'Clock',
    get desc() {
      return t('时区与 NTP')
    },
  },
  {
    key: 'power',
    get title() {
      return t('电源')
    },
    icon: 'SwitchButton',
    get desc() {
      return t('锁定 / 重启 / 关机')
    },
  },
  {
    key: 'update',
    get title() {
      return t('系统更新')
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
      return t('检测仓库并下载安装')
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
