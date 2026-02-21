import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/lxc-dev-manager/',
  title: 'lxc-dev-manager',
  description: 'Manage LXC containers for local development',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/lxc-dev-manager/logo.svg' }]
  ],

  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/about' },
      { text: 'Commands', link: '/reference/commands/' },
      { text: 'Reference', link: '/reference/configuration' }
    ],

    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'About', link: '/guide/about' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'LXC Setup', link: '/guide/setup' },
          { text: 'Workflow', link: '/guide/workflow' }
        ]
      },
      {
        text: 'Commands',
        items: [
          { text: 'Overview', link: '/reference/commands/' },
          { text: 'Project', link: '/reference/commands/project' },
          { text: 'Container', link: '/reference/commands/container' },
          { text: 'Snapshot', link: '/reference/commands/snapshot' },
          { text: 'Image', link: '/reference/commands/image' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'Configuration', link: '/reference/configuration' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/pierre-yves-mathieu/lxc-dev-manager' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright 2024-present'
    },

    search: {
      provider: 'local'
    }
  }
})
