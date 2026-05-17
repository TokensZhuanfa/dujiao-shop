import { defineConfig } from 'vitepress'

// GitHub Pages 项目站点: https://TokensZhuanfa.github.io/Dujiao-Shop/
export default defineConfig({
  base: '/Dujiao-Shop/',
  lang: 'zh-CN',
  title: 'dujiao-shop',
  description: '自托管自动发卡 / 卡密商店 · 文档',

  lastUpdated: false,
  cleanUrls: true,
  ignoreDeadLinks: true,

  head: [
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/Dujiao-Shop/logo.svg' }],
  ],

  themeConfig: {
    siteTitle: 'dujiao-shop',
    logo: { src: '/logo.svg', alt: 'dujiao-shop' },

    nav: [
      { text: '指南', link: '/intro/about', activeMatch: '^/(intro|config|deploy|guide|payment|ops|community|api|services|sponsor)' },
      { text: 'Demo', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
      { text: 'GitHub', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
    ],

    // 全局 sidebar (参考 dujiao-next.com 10 大分组)
    sidebar: [
      {
        text: '简介',
        collapsed: false,
        items: [
          { text: '关于 dujiao-shop', link: '/intro/about' },
          { text: '环境要求', link: '/intro/requirements' },
          { text: '更新日志', link: '/intro/changelog' },
          { text: '术语统一表', link: '/intro/glossary' },
          { text: '开源仓库与贡献', link: '/intro/repos' },
        ],
      },
      {
        text: '配置',
        collapsed: false,
        items: [
          { text: 'config.yml 详细说明', link: '/config/yaml' },
        ],
      },
      {
        text: '部署',
        collapsed: false,
        items: [
          { text: '部署总览', link: '/deploy/overview' },
          { text: '单二进制部署 (推荐小白)', link: '/deploy/single-binary' },
          { text: '手动部署 (源码)', link: '/deploy/manual' },
          { text: 'Docker Compose 部署', link: '/deploy/docker-compose' },
          { text: '宝塔面板部署', link: '/deploy/baota' },
        ],
      },
      {
        text: '使用指南',
        collapsed: false,
        items: [
          { text: '后台管理入门', link: '/guide/admin-getting-started' },
          { text: '卡密管理', link: '/guide/cards' },
          { text: 'Codex 号池', link: '/guide/codex-pool' },
          { text: '钱包与礼品卡', link: '/guide/wallet-giftcard' },
          { text: '优惠券与活动价', link: '/guide/coupons' },
          { text: '会员等级', link: '/guide/member-levels' },
          { text: '分销推广', link: '/guide/affiliate' },
          { text: '通知中心配置', link: '/guide/notifications' },
          { text: '安全最佳实践', link: '/guide/security' },
          { text: '常见问题 FAQ', link: '/guide/faq' },
        ],
      },
      {
        text: '支付',
        collapsed: false,
        items: [
          { text: '支付配置与回调指南', link: '/payment/setup' },
        ],
      },
      {
        text: '部署运维',
        collapsed: false,
        items: [
          { text: '升级与迁移', link: '/ops/upgrade-migration' },
          { text: '备份与恢复', link: '/ops/backup-restore' },
          { text: '故障排查', link: '/ops/troubleshooting' },
        ],
      },
      {
        text: '社区',
        collapsed: true,
        items: [
          { text: '社区共享项目', link: '/community/shared-projects' },
        ],
      },
      {
        text: 'API 集成',
        collapsed: true,
        items: [
          { text: 'User 前台 API 文档', link: '/api/user' },
          { text: '站点对接说明', link: '/api/site-integration' },
          { text: '站点对接 API 文档', link: '/api/site-integration-api' },
        ],
      },
      {
        text: '官方服务',
        collapsed: true,
        items: [
          { text: '官方服务说明', link: '/services/official' },
          { text: 'Telegram Bot 服务介绍', link: '/services/telegram-bot' },
        ],
      },
      {
        text: '赞助',
        collapsed: true,
        items: [
          { text: '成为赞助商', link: '/sponsor/become' },
          { text: '赞助商名单', link: '/sponsor/silver-list' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
    ],

    footer: {
      message: 'MIT License',
      copyright: '© 2026 TokensZhuanfa',
    },

    search: {
      provider: 'local',
      options: {
        locales: {
          root: {
            translations: {
              button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
              modal: {
                noResultsText: '无搜索结果',
                resetButtonTitle: '清空',
                footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' },
              },
            },
          },
        },
      },
    },

    docFooter: { prev: '上一篇', next: '下一篇' },
    outline: { label: '本页目录', level: [2, 3] },
    lastUpdatedText: '最后更新',
    darkModeSwitchLabel: '主题',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '回到顶部',
  },
})
