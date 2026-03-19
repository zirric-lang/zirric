module.exports = {
  title: "Zirric Language Documentation",
  url: "https://zirric.knabel.dev",
  logo: {
    light: "/assets/images/zirric.svg",
    dark: "/assets/images/zirric.svg",
    alt: "Zirric Logo",
    href: "/"
  },
  favicon: "/assets/favicon.ico",
  src: "../../docs",
  out: "./site",
  layout: {
    spa: true,
    header: {
      enabled: true
    },
    sidebar: {
      collapsible: true,
      defaultCollapsed: false
    },
    optionsMenu: {
      position: "header",
      components: {
        search: true,
        themeSwitch: true,
        sponsor: null
      }
    },
    footer: {
      style: "minimal",
      content: "© 2026 Zirric. Built with [docmd](https://docmd.io)."
    }
  },
  theme: {
    name: "ruby",
    appearance: "system",
    codeHighlight: true,
    customCss: []
  },
  minify: true,
  autoTitleFromH1: true,
  copyCode: true,
  pageNavigation: true,
  customJs: [],
  editLink: {
    enabled: true,
    baseUrl: "https://code.knabel.dev/zirric-lang/zirric/_edit/main/docs",
    text: "Edit this page"
  },
  plugins: {
    seo: {
      defaultDescription: "Documentation built with docmd.",
      openGraph: {
        defaultImage: "/assets/images/logo@512x.png"
      }
    },
    sitemap: {
      defaultChangefreq: "monthly",
      defaultPriority: 0.8
    },
    llms: {},
    "custom-header": {
      header: [
        "<script defer src=\"https://uma.knabel.dev/umami\" data-website-id=\"cb95e6bc-e06f-4398-9dad-bf39ccdb99b9\" data-domains=\"zirric.knabel.dev\"></script>",
        "<link rel=\"icon\" type=\"image/svg+xml\" href=\"/assets/images/favicon/favicon.svg\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"96x96\" href=\"/assets/images/favicon/favicon-96x96.png\">",
        "<link rel=\"apple-touch-icon\" href=\"/assets/images/favicon/apple-touch-icon.png\">",
        "<link rel=\"manifest\" href=\"/assets/images/favicon/site.webmanifest\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"192x192\" href=\"/assets/images/favicon/web-app-manifest-192x192.png\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"512x512\" href=\"/assets/images/favicon/web-app-manifest-512x512.png\">"
      ]
    }
  },
  redirects: [],
  notFound: {
    title: 'Page Not Found', 
    content: 'Oops! This page has moved.'
  },
  navigation: [
    {
      title: "Overview",
      path: "/overview",
      icon: "layout-list"
    },
    {
      title: "Guides",
      path: "/guides",
      icon: "book-open",
      collapsible: true,
      children: [
        {
          title: "Getting Started",
          path: "/guides/getting-started",
          icon: "rocket"
        },
        {
          title: "Installation",
          path: "/guides/installation",
          icon: "hard-drive"
        },
        {
          title: "Styleguide",
          path: "/guides/styleguide",
          icon: "paintbrush"
        }
      ]
    },
    {
      title: "Tooling",
      path: "/tooling",
      icon: "wrench",
      collapsible: true,
      children: [
        {
          title: "Editor Support",
          path: "/tooling/editor-support",
          icon: "dna"
        },
        {
          title: "Package Manager",
          path: "/tooling/package-manager",
          icon: "box"
        },
        {
          title: "Compiler",
          path: "/tooling/compiler",
          icon: "cpu"
        }
      ]
    },
    {
      title: "Standard Library",
      path: "/stdlib",
      icon: "library",
      collapsible: true,
      children: [
        {
          title: "Prelude",
          path: "/stdlib/prelude",
          icon: "sparkles"
        },
        {
          title: "Future",
          path: "/stdlib/future",
          icon: "flask-conical",
          collapsible: true,
          children: [
            {
              title: "Prelude",
              path: "/stdlib/future/prelude",
              icon: "sparkles"
            },
            {
              title: "Reflect",
              path: "/stdlib/future/reflect",
              icon: "scan"
            },
            {
              title: "Cave",
              path: "/stdlib/future/cave",
              icon: "package"
            },
            {
              title: "Tasks",
              path: "/stdlib/future/tasks",
              icon: "check-square"
            }
          ]
        }
      ]
    },
    {
      title: "Changelog",
      path: "/changelog",
      icon: "history",
      collapsible: true,
      children: [
        {
          title: "v0.1.0-next",
          path: "/changelog/v0.1.0",
          icon: "git-branch"
        },
        {
          title: "v0.0.1",
          path: "/changelog/v0.0.1",
          icon: "tag"
        }
      ]
    },
    {
      title: "Proposals",
      path: "/proposals",
      icon: "circle",
      collapsible: true,
      children: [
        {
          icon: "circle-check",
          title: "ZE-01 Language",
          path: "/proposals/ZE-001-base-language"
        },
        {
          icon: "circle-dot",
          title: "ZE-02 Cavefile",
          path: "/proposals/ZE-002-the-cavefile"
        },
        {
          icon: "circle-x",
          title: "ZE-03 Named Params",
          path: "/proposals/ZE-003-named-data-construction"
        },
        {
          icon: "circle",
          title: "ZE-04 Variadic",
          path: "/proposals/ZE-004-Variadic-Arguments"
        },
        {
          icon: "circle",
          title: "ZE-05 Mixins",
          path: "/proposals/ZE-005-Mixin-Type-Declarations"
        },
        {
          icon: "circle",
          title: "ZE-06 Parsing",
          path: "/proposals/ZE-006-Attribute-Based-Parsing-System"
        },
        {
          icon: "circle",
          title: "ZE-07 Attribute Binding",
          path: "/proposals/ZE-007-attribute-binding"
        },
        {
          icon: "circle-dot",
          title: "ZE-08 Errors",
          path: "/proposals/ZE-008-error-handling"
        },
        {
          icon: "circle-dot",
          title: "ZE-09 Option",
          path: "/proposals/ZE-009-option-values"
        },
        {
          icon: "circle-dot",
          title: "ZE-10 Iterable",
          path: "/proposals/ZE-010-iterable"
        },
        {
          icon: "circle-check",
          title: "ZE-11 Zirric CLI",
          path: "/proposals/ZE-011-zirric-cli"
        },
        {
          icon: "circle-x",
          title: "ZE-12 Type and Returns Sugar",
          path: "/proposals/ZE-012-type-and-returns-sugar"
        },
        {
          icon: "circle-check",
          title: "ZE-13 Mutability and Constants",
          path: "/proposals/ZE-013-mutability-and-constants"
        },
        {
          icon: "circle-check",
          title: "ZE-14 Attribute and Declaration Keywords",
          path: "/proposals/ZE-014-attribute-and-declaration-keywords"
        },
        {
          icon: "circle",
          title: "ZE-15 Extern Type Constructors",
          path: "/proposals/ZE-015-extern-type-constructors"
        },
        {
          icon: "circle-check",
          title: "ZE-16 Closure Syntax",
          path: "/proposals/ZE-016-closure-syntax"
        },
        {
          icon: "circle-check",
          title: "ZE-17 Type Hints",
          path: "/proposals/ZE-017-type-hints"
        },
        {
          icon: "circle",
          title: "ZE-18 I/O, Fmt, OS",
          path: "/proposals/ZE-018-io-fmt-os"
        },
        {
          icon: "circle",
          title: "ZE-19 Result & Option Sugar",
          path: "/proposals/ZE-019-result-and-option-sugar"
        }
      ]
    },
    {
      title: "Specification",
      path: "/specification",
      icon: "pencil-ruler",
      collapsible: true,
      children: [
        {
          title: "Syntax",
          path: "/specification/syntax",
          icon: "code"
        },
        {
          title: "Declarations",
          path: "/specification/declarations",
          icon: "file-text"
        },
        {
          title: "Expressions",
          path: "/specification/expressions",
          icon: "braces"
        },
        {
          title: "Type System",
          path: "/specification/typesystem",
          icon: "shapes"
        }
      ]
    },
    {
      title: "Repository",
      path: "https://code.knabel.dev/zirric-lang/zirric",
      icon: "git-graph",
      external: true
    },
    {
      title: "Tree-Sitter-Zirric",
      path: "https://code.knabel.dev/zirric-lang/tree-sitter-zirric",
      icon: "git-branch",
      external: true
    }
  ]
};
