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
      { text: '首页', link: '/' },
      {
        text: '安装部署',
        items: [
          { text: 'Docker Compose', link: '/install/docker' },
          { text: '宝塔面板', link: '/install/baota' },
          { text: '二进制 (轻量)', link: '/install/binary' },
          { text: '源码编译', link: '/install/source' },
        ],
      },
      {
        text: '配置参考',
        items: [
          { text: 'config.yml 详解', link: '/reference/config' },
        ],
      },
      {
        text: '后台管理',
        items: [
          { text: '使用速览', link: '/admin/usage' },
          { text: '支付通道接入', link: '/admin/payment' },
        ],
      },
      {
        text: '运维',
        items: [
          { text: '备份恢复', link: '/ops/backup-restore' },
          { text: '故障排查', link: '/ops/troubleshooting' },
          { text: '安全加固', link: '/ops/security' },
        ],
      },
      {
        text: '二次开发',
        items: [
          { text: '项目结构', link: '/customization/architecture' },
          { text: 'Codex 号池', link: '/customization/codex-pool' },
          { text: '上游同步', link: '/customization/upstream-sync' },
        ],
      },
      { text: 'FAQ', link: '/faq' },
      { text: '发布', link: '/release' },
      { text: 'GitHub', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
    ],

    sidebar: {
      '/install/': [
        {
          text: '安装部署',
          items: [
            { text: 'Docker Compose (推荐)', link: '/install/docker' },
            { text: '宝塔面板', link: '/install/baota' },
            { text: '二进制 (轻量)', link: '/install/binary' },
            { text: '源码编译', link: '/install/source' },
          ],
        },
        {
          text: '安装后',
          items: [
            { text: 'config.yml 详解', link: '/reference/config' },
            { text: '后台使用', link: '/admin/usage' },
            { text: '支付通道接入', link: '/admin/payment' },
          ],
        },
      ],
      '/reference/': [
        {
          text: '配置参考',
          items: [
            { text: 'config.yml 详解', link: '/reference/config' },
          ],
        },
      ],
      '/admin/': [
        {
          text: '后台管理',
          items: [
            { text: '使用速览', link: '/admin/usage' },
            { text: '支付通道接入', link: '/admin/payment' },
          ],
        },
      ],
      '/ops/': [
        {
          text: '运维',
          items: [
            { text: '备份恢复', link: '/ops/backup-restore' },
            { text: '故障排查', link: '/ops/troubleshooting' },
            { text: '安全加固', link: '/ops/security' },
          ],
        },
      ],
      '/customization/': [
        {
          text: '二次开发',
          items: [
            { text: '项目结构', link: '/customization/architecture' },
            { text: 'Codex 号池', link: '/customization/codex-pool' },
            { text: '上游同步', link: '/customization/upstream-sync' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/TokensZhuanfa/Dujiao-Shop' },
    ],

    footer: {
      message: '基于 <a href="https://github.com/dujiao-next">dujiao-next</a> 上游 fork · MIT License',
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
