// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    'intro',
    'quickstart',
    {
      type: 'category',
      label: 'Concepts',
      collapsed: false,
      items: [
        'concepts/overview',
        'concepts/projects-environments',
        'concepts/principals',
        'concepts/sessions',
        'concepts/registration',
        'concepts/mfa',
        'concepts/webhooks-hooks',
        'concepts/oidc-federation',
        'concepts/signing-keys',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      collapsed: false,
      items: [
        'guides/sdk-quickstart',
        'guides/auth-flows',
        'guides/passwordless',
        'guides/oauth-social',
        'guides/mfa',
        'guides/admin-config',
        'guides/notifications',
        'guides/user-management',
        'guides/machine-identity',
        'guides/enterprise-sso',
        'guides/security-controls',
        'guides/import-export',
        'guides/test-mode',
      ],
    },
    {
      type: 'category',
      label: 'REST API',
      items: [
        'rest-api/overview',
        'rest-api/authentication',
        'rest-api/flows',
        'rest-api/runtime',
        'rest-api/admin',
        'rest-api/errors',
      ],
    },
    {
      type: 'category',
      label: 'SDK',
      items: ['sdk/typescript', 'sdk/go'],
    },
    {
      type: 'category',
      label: 'Self-hosting',
      items: [
        'self-hosting/docker',
        'self-hosting/configuration',
        'self-hosting/operator',
        'self-hosting/deployment',
        'self-hosting/observability',
        'self-hosting/migrations',
      ],
    },
  ],
};

module.exports = sidebars;
