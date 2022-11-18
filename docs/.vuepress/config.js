require("dotenv").config();

const metaTagsValue = {
  author: "InjectiveProtocol",
  url: "https://docs.injective.network",
  shortName: "Injective Docs",
  twitterHandle: "@InjectiveLabs",
  ogImage: "/meta_img.jpg",
};

const metaTags = [
  ["meta", { name: "twitter:card", content: metaTagsValue.ogImage }],
  ["meta", { name: "twitter:site", content: metaTagsValue.twitterHandle }],
  ["meta", { name: "twitter:creator", content: metaTagsValue.twitterHandle }],
  ["meta", { property: "og:type", content: "website" }],
  ["meta", { property: "og:title", content: metaTagsValue.shortName }],
  ["meta", { property: "og:site_name", content: metaTagsValue.shortName }],
  ["meta", { name: "og:url", content: metaTagsValue.url }],
  ["meta", { name: "og:image", content: metaTagsValue.ogImage }],
];

module.exports = {
  theme: "cosmos",
  title: "Injective Chain Documentation",
  image: "/meta_img.jpg",
  locales: {
    "/": {
      lang: "en-US",
    },
  },
  markdown: {
    extendMarkdown: (md) => {
      md.use(require("markdown-it-katex"));
    },
  },
  head: [
    ...metaTags,
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.5.1/katex.min.css",
      },
    ],
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://cdn.jsdelivr.net/github-markdown-css/2.2.1/github-markdown.css",
      },
    ],
    [
      "script",
      {
        async: true,
        src:
          "https://www.googletagmanager.com/gtag/js?id=G-" +
          process.env.APP_GOOGLE_ANALYTICS_KEY,
      },
    ],
    [
      "script",
      {},
      [
        `window.dataLayer = window.dataLayer || [];\nfunction gtag(){dataLayer.push(arguments);}\ngtag('js', new Date());\ngtag('config', 'G-${process.env.APP_GOOGLE_ANALYTICS_KEY}');`,
      ],
    ],
  ],
  base: process.env.VUEPRESS_BASE || "/",
  themeConfig: {
    repo: "InjectiveLabs/injective-core",
    docsRepo: "InjectiveLabs/injective-core",
    docsBranch: "dev",
    docsDir: "docs",
    editLinks: true,
    custom: true,
    defaultImage: "/meta_img.jpg",
    logo: {
      src: "/logo.png",
    },
    topbar: {
      banner: false,
    },
    sidebar: {
      auto: false,
      nav: [
        {
          title: "About Injective",
          children: [
            {
              title: "Introduction",
              directory: true,
              path: "/intro",
            },
            {
              title: "Glossary",
              path: "/glossary/",
            },
            {
              title: "Injective Ecosystem",
              path: "https://injective.com/ecosystem",
            },
          ],
        },
        {
          title: "For Users",
          children: [
            {
              title: "Basic Concepts",
              directory: true,
              path: "/concepts",
            },
            {
              title: "Chain Modules",
              directory: true,
              path: "/modules",
            },
            {
              title: "Injective Hub",
              directory: true,
              path: "/hub",
            },
          ],
        },
        {
          title: "For Developers",
          children: [
            {
              title: "Technical Concepts",
              directory: true,
              path: "/tech-concepts",
            },
            {
              title: "Tools",
              directory: true,
              path: "/tools",
            },
            {
              title: "Building DApps With CosmWasm",
              directory: true,
              path: "/cosmwasm-dapps",
            },
            {
              title: "Building Orderbook Exchanges",
              directory: true,
              path: "/exchange",
            },
            {
              title: "Networks",
              directory: true,
              path: "/networks",
            },
          ],
        },
        {
          title: "For Traders",
          children: [
            {
              title: "Trader API Documentation",
              path: "https://api.injective.exchange/",
            },
          ],
        },
        {
          title: "For Validators",
          children: [
            {
              title: "Mainnet",
              children: [
                {
                  title: "Canonical Chain Upgrade",
                  directory: true,
                  path: "/guides/mainnet/canonical-chain-upgrade",
                  children: [
                    {
                      title: "Upgrade Instructions",
                      directory: false,
                      path: "/guides/mainnet/canonical-chain-upgrade",
                    },
                    {
                      title: "Upgrade to 10002-rc1",
                      directory: false,
                      path: "/guides/mainnet/canonical-10002-rc1",
                    },
                    {
                      title: "Upgrade to 10002-rc2",
                      directory: false,
                      path: "/guides/mainnet/canonical-10002-rc2",
                    },
                    {
                      title: "Upgrade to 10003-rc1",
                      directory: false,
                      path: "/guides/mainnet/canonical-10003-rc1",
                    },
                    {
                      title: "Upgrade to 10004-rc1",
                      directory: false,
                      path: "/guides/mainnet/canonical-10004-rc1",
                    },
                    {
                      title: "Upgrade to 10004-rc1-patch",
                      directory: false,
                      path: "/guides/mainnet/canonical-10004-rc1-patch",
                    },
                    {
                      title: "Upgrade to 10005-rc1",
                      directory: false,
                      path: "/guides/mainnet/canonical-10005-rc1",
                    },
                    {
                      title: "Upgrade to 10006-rc1",
                      directory: false,
                      path: "/guides/mainnet/canonical-10006-rc1",
                    },
                  ],
                },
                {
                  title: "Becoming a Validator",
                  directory: false,
                  path: "/guides/mainnet/becoming-a-validator",
                },
                {
                  title: "Setup Peggo Orchestrator",
                  directory: false,
                  path: "/guides/mainnet/peggo",
                },
              ],
            },
            {
              title: "Testnet",
              children: [
                {
                  title: "Becoming a Validator",
                  directory: false,
                  path: "/guides/testnet/becoming-a-validator",
                },
                {
                  title: "Setup Peggo Orchestrator",
                  directory: false,
                  path: "/guides/testnet/peggo",
                },
              ],
            },
          ],
        },
        {
          title: "Resources",
          children: [
            {
              title: "Injective REST API Spec",
              path: "https://lcd.injective.network/swagger/",
            },
            {
              title: "Injective Explorer",
              path: "https://explorer.injective.network/",
            },
            {
              title: "Mintscan",
              path: "https://www.mintscan.io/injective",
            },
            {
              title: "Commonwealth Discussion Forum",
              path: "https://gov.injective.network/",
            },
          ],
        },
      ],
    },
    gutter: {
      title: "Help & Support",
      chat: {
        title: "Developer Chat",
        text: "Chat with Injective developers on Discord.",
        url: "https://discord.gg/injective",
        bg: "linear-gradient(103.75deg, #1B1E36 0%, #22253F 100%)",
      },
      github: {
        title: "Found an Issue?",
        text: "Help us improve this page by suggesting edits on GitHub.",
        url: "https://docs.google.com/forms/d/e/1FAIpQLSc9LYBdSOd28iwm4CdWsbIt0T6jwzRdlkdrGgfqNu5smWlCkg/viewform",
        bg: "#F8F9FC",
      },
      forum: {
        title: "Injective Forum",
        text: "Join the Injective Forum to learn more.",
        url: "https://gov.injective.network/",
        bg: "linear-gradient(225deg, #46509F -1.08%, #2F3564 95.88%)",
        logo: "cosmos",
      },
    },
    footer: {
      logo: "/logo.png",
      textLink: {
        text: "injective.com",
        url: "https://injective.com",
      },
      services: [
        {
          service: "github",
          url: "https://github.com/InjectiveLabs/injective-core",
        },
        {
          service: "twitter",
          url: "https://twitter.com/Injective_",
        },
        {
          service: "linkedin",
          url: "https://www.linkedin.com/company/injective-protocol",
        },
        {
          service: "medium",
          url: "https://injectiveprotocol.medium.com",
        },
      ],
      smallprint:
        "This website is maintained by [Injective](https://injective.com).",
      links: [],
    },
  },
};
