import { defineConfig } from 'vitepress'

// GitHub Pages 项目站点: https://TokensZhuanfa.github.io/Dujiao-Shop/
export default defineConfig({
  base: '/Dujiao-Shop/',
  lang: 'zh-CN',
  title: 'dujiao-shop',
  description: '自托管自动发卡 / 卡密商店 · 文档',

  lastUpdated: true,
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
      { text: '指南', link: '/install/docker', activeMatch: '^/(install|reference|admin|ops|customization|faq)' },
      { text: 'FAQ', link: '/faq' },
      { text: '发布', link: '/release' },
      { text: 'GitHub', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
    ],

    // 全局 sidebar (array 形式 → 所有非首页页面都看到同一份)
    sidebar: [
      {
        text: '简介',
        collapsed: false,
        items: [
          { text: '快速开始', link: '/install/docker' },
          { text: '功能特性', link: '/' },
          { text: '常见问题 FAQ', link: '/faq' },
        ],
      },
      {
        text: '部署',
        collapsed: false,
        items: [
          { text: 'Docker Compose (推荐)', link: '/install/docker' },
          { text: '宝塔面板', link: '/install/baota' },
          { text: '单二进制 (轻量)', link: '/install/binary' },
          { text: '源码编译', link: '/install/source' },
        ],
      },
      {
        text: '配置',
        collapsed: false,
        items: [
          { text: 'config.yml 详解', link: '/reference/config' },
        ],
      },
      {
        text: '使用指南',
        collapsed: false,
        items: [
          { text: '后台管理速览', link: '/admin/usage' },
          { text: '支付通道接入', link: '/admin/payment' },
          { text: 'Codex 号池', link: '/customization/codex-pool' },
        ],
      },
      {
        text: '部署运维',
        collapsed: false,
        items: [
          { text: '备份与恢复', link: '/ops/backup-restore' },
          { text: '故障排查', link: '/ops/troubleshooting' },
          { text: '安全加固', link: '/ops/security' },
        ],
      },
      {
        text: '二次开发',
        collapsed: true,
        items: [
          { text: '项目结构', link: '/customization/architecture' },
          { text: '上游同步', link: '/customization/upstream-sync' },
          { text: '发布流程', link: '/release' },
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
