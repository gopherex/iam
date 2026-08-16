// @ts-check
// Docusaurus site for the gopherex IAM authentication service.
// Deployed to GitHub Pages at https://gopherex.github.io/iam/.

const { themes } = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'IAM',
  tagline: 'Headless authentication & identity — multi-tenant, self-hostable',
  favicon: 'img/favicon.svg',

  url: 'https://gopherex.github.io',
  baseUrl: '/iam/',

  organizationName: 'gopherex',
  projectName: 'iam',
  deploymentBranch: 'gh-pages',
  trailingSlash: false,

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },
  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: '/', // docs are the site root
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/gopherex/iam/edit/master/website/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      colorMode: {
        respectPrefersColorScheme: true,
      },
      navbar: {
        title: 'IAM',
        logo: {
          alt: 'IAM',
          src: 'img/logo.svg',
        },
        items: [
          { type: 'docSidebar', sidebarId: 'docs', position: 'left', label: 'Docs' },
          { to: '/rest-api/overview', label: 'REST API', position: 'left' },
          { to: '/sdk/typescript', label: 'SDK', position: 'left' },
          {
            href: 'https://github.com/gopherex/iam',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              { label: 'Introduction', to: '/' },
              { label: 'Quickstart', to: '/quickstart' },
              { label: 'Concepts', to: '/concepts/overview' },
            ],
          },
          {
            title: 'Reference',
            items: [
              { label: 'REST API', to: '/rest-api/overview' },
              { label: 'TypeScript SDK', to: '/sdk/typescript' },
              { label: 'Self-hosting', to: '/self-hosting/docker' },
            ],
          },
          {
            title: 'More',
            items: [
              { label: 'GitHub', href: 'https://github.com/gopherex/iam' },
              { label: 'OpenAPI spec', href: 'https://github.com/gopherex/iam/blob/master/openapi/openapi.yaml' },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} gopherex IAM.`,
      },
      prism: {
        theme: themes.github,
        darkTheme: themes.dracula,
        additionalLanguages: ['bash', 'json', 'go', 'yaml'],
      },
    }),
};

module.exports = config;
