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
    appearance: "dark",
    codeHighlight: true,
    customCss: ['/assets/styles/overrides.css']
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
      defaultDescription: "Language documentation of Zirric.",
      openGraph: {
        defaultImage: "/assets/images/logo@512x.png"
      }
    },
    sitemap: {
      defaultChangefreq: "monthly",
      defaultPriority: 0.8
    },
    llms: {},
    ai: false,
    "./plugins/custom-header-plugin.js": {
      headScriptsHtml: [
        "<script defer src=\"https://uma.knabel.dev/umami\" data-website-id=\"773650f2-09b3-4afa-bdcb-06e9e2957fd4\" data-domains=\"zirric.knabel.dev\"></script>",
      ],
      metaTags: [
        "<link rel=\"icon\" type=\"image/svg+xml\" href=\"/assets/images/favicon/favicon.svg\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"96x96\" href=\"/assets/images/favicon/favicon-96x96.png\">",
        "<link rel=\"apple-touch-icon\" href=\"/assets/images/favicon/apple-touch-icon.png\">",
        "<link rel=\"manifest\" href=\"/assets/images/favicon/site.webmanifest\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"192x192\" href=\"/assets/images/favicon/web-app-manifest-192x192.png\">",
        "<link rel=\"icon\" type=\"image/png\" sizes=\"512x512\" href=\"/assets/images/favicon/web-app-manifest-512x512.png\">"
      ],
    }
  },
  redirects: {
    "/tooling/editor-support": "/tooling/editor-configuration",
    "/tooling/formatter": "/tooling/code-formatter"
  },
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
          title: "Zirric CLI",
          path: "/tooling/zirric-cli",
          icon: "terminal"
        },
        {
          title: "Editor Configuration",
          path: "/tooling/editor-configuration",
          icon: "settings"
        },
        {
          title: "Compiler",
          path: "/tooling/compiler",
          icon: "cpu"
        },
        {
          title: "Language Server",
          path: "/tooling/language-server",
          icon: "server"
        },
        {
          title: "Code Formatter",
          path: "/tooling/code-formatter",
          icon: "align-left"
        },
        {
          title: "Package Manager",
          path: "/tooling/package-manager",
          icon: "box"
        },
        {
          title: "Tree Sitter",
          path: "https://code.knabel.dev/zirric-lang/tree-sitter-zirric",
          icon: "dna",
          external: true
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
          title: "Core",
          icon: "sparkles",
          collapsible: true,
          children: [
            {
              title: "Prelude",
              path: "/stdlib/prelude",
              icon: "sparkles"
            },
            {
              title: "Options",
              path: "/stdlib/options",
              icon: "circle-help"
            },
            {
              title: "Results",
              path: "/stdlib/results",
              icon: "circle-check"
            },
            {
              title: "Errors",
              path: "/stdlib/errors",
              icon: "circle-alert"
            }
          ]
        },
        {
          title: "Data",
          icon: "database",
          collapsible: true,
          children: [
            {
              title: "Arrays",
              path: "/stdlib/arrays",
              icon: "list"
            },
            {
              title: "Dicts",
              path: "/stdlib/dicts",
              icon: "table-2"
            },
            {
              title: "Strings",
              path: "/stdlib/strings",
              icon: "type"
            },
            {
              title: "Bytes",
              path: "/stdlib/bytes",
              icon: "binary"
            },
            {
              title: "Ranges",
              path: "/stdlib/ranges",
              icon: "ruler"
            },
            {
              title: "Math",
              path: "/stdlib/math",
              icon: "sigma"
            },
            {
              title: "Fun",
              path: "/stdlib/fun",
              icon: "workflow"
            }
          ]
        },
        {
          title: "System",
          icon: "monitor",
          collapsible: true,
          children: [
            {
              title: "OS",
              path: "/stdlib/os",
              icon: "monitor"
            },
            {
              title: "IO",
              path: "/stdlib/io",
              icon: "arrow-left-right"
            },
            {
              title: "Fmt",
              path: "/stdlib/fmt",
              icon: "text"
            },
            {
              title: "FS",
              path: "/stdlib/fs",
              icon: "folder"
            },
            {
              title: "Paths",
              path: "/stdlib/paths",
              icon: "route"
            },
            {
              title: "Clock",
              path: "/stdlib/clock",
              icon: "clock"
            },
            {
              title: "Time",
              path: "/stdlib/time",
              icon: "calendar"
            },
            {
              title: "Random",
              path: "/stdlib/random",
              icon: "dices"
            },
            {
              title: "Scripts",
              path: "/stdlib/scripts",
              icon: "scroll"
            }
          ]
        },
        {
          title: "Serialization",
          icon: "arrow-right-left",
          collapsible: true,
          children: [
            {
              title: "Coding",
              path: "/stdlib/coding",
              icon: "arrow-right-left"
            },
            {
              title: "JSON",
              path: "/stdlib/json",
              icon: "braces"
            },
            {
              title: "YAML",
              path: "/stdlib/yaml",
              icon: "file-code"
            }
          ]
        },
        {
          title: "Reflection",
          icon: "scan",
          collapsible: true,
          children: [
            {
              title: "Reflect",
              path: "/stdlib/reflect",
              icon: "scan"
            },
            {
              title: "Reflect.Packages",
              path: "/stdlib/reflect/packages",
              icon: "boxes"
            }
          ]
        },
        {
          title: "Testing",
          icon: "flask-conical",
          collapsible: true,
          children: [
            {
              title: "Tests",
              path: "/stdlib/tests",
              icon: "flask-conical"
            },
            {
              title: "Tests.Assert",
              path: "/stdlib/tests/assert",
              icon: "check-check"
            },
            {
              title: "Tests.Runner",
              path: "/stdlib/tests/runner",
              icon: "play"
            },
            {
              title: "Tests.TAP",
              path: "/stdlib/tests/tap",
              icon: "receipt"
            }
          ]
        },
        {
          title: "Manifests",
          icon: "package",
          collapsible: true,
          children: [
            {
              title: "Cave",
              path: "/stdlib/cave",
              icon: "package"
            },
            {
              title: "Tasks",
              path: "/stdlib/tasks",
              icon: "check-square"
            }
          ]
        },
        {
          title: "Future",
          icon: "circle-dashed",
          collapsible: true,
          children: [
            {
              title: "Future",
              path: "/stdlib/future",
              icon: "flask-conical"
            },
            {
              title: "Future.Prelude",
              path: "/stdlib/future/prelude",
              icon: "sparkles"
            },
            {
              title: "Future.Reflect",
              path: "/stdlib/future/reflect",
              icon: "scan"
            }
          ]
        }
      ]
    },
    {
      title: "External Packages",
      icon: "blocks",
      collapsible: true,
      children: [
        {
          title: "Colors",
          path: "https://zirric-colors.knabel.dev",
          icon: "palette",
          external: true
        },
        {
          title: "UI",
          path: "https://zirric-ui.knabel.dev",
          icon: "app-window",
          external: true
        },
        {
          title: "HTML",
          path: "https://code.knabel.dev/zirric-lang/html",
          icon: "code-xml",
          external: true
        },
        {
          title: "MD",
          path: "https://code.knabel.dev/zirric-lang/md",
          icon: "file-text",
          external: true
        },
        {
          title: "Term",
          path: "https://code.knabel.dev/zirric-lang/term",
          icon: "terminal",
          external: true
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
          title: "v0.1.0",
          path: "/changelog/v0.1.0",
          icon: "tag"
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
          icon: "circle-check",
          title: "ZE-02 Cavefile",
          path: "/proposals/ZE-002-the-cavefile"
        },
        {
          icon: "circle-x",
          title: "ZE-03 Named Params",
          path: "/proposals/ZE-003-named-data-construction"
        },
        {
          icon: "circle-x",
          title: "ZE-04 Variadic Attributes",
          path: "/proposals/ZE-004-Variadic-Attributes"
        },
        {
          icon: "circle",
          title: "ZE-05 Mixins",
          path: "/proposals/ZE-005-Mixin-Type-Declarations"
        },
        {
          icon: "circle-x",
          title: "ZE-06 Parsing",
          path: "/proposals/ZE-006-Attribute-Based-Parsing-System"
        },
        {
          icon: "circle",
          title: "ZE-07 Attribute Binding",
          path: "/proposals/ZE-007-attribute-binding"
        },
        {
          icon: "circle-check",
          title: "ZE-08 Errors",
          path: "/proposals/ZE-008-error-handling"
        },
        {
          icon: "circle-check",
          title: "ZE-09 Option",
          path: "/proposals/ZE-009-option-values"
        },
        {
          icon: "circle-check",
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
          icon: "circle-check",
          title: "ZE-18 I/O, Fmt, OS",
          path: "/proposals/ZE-018-io-fmt-os"
        },
        {
          icon: "circle-check",
          title: "ZE-19 Result & Option Sugar",
          path: "/proposals/ZE-019-result-and-option-sugar"
        },
        {
          icon: "circle-check",
          title: "ZE-20 Standard Library",
          path: "/proposals/ZE-020-standard-library"
        },
        {
          icon: "circle-check",
          title: "ZE-21 Encoding and Decoding",
          path: "/proposals/ZE-021-encoding-and-decoding"
        },
        {
          icon: "circle-check",
          title: "ZE-22 Static Checks",
          path: "/proposals/ZE-022-static-checks"
        },
        {
          icon: "circle-check",
          title: "ZE-23 Code Formatting",
          path: "/proposals/ZE-023-code-formatting"
        },
        {
          icon: "circle-check",
          title: "ZE-24 Module Names",
          path: "/proposals/ZE-024-qualified-module-names"
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
    }
  ]
};
