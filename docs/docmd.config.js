module.exports = {
  siteTitle: 'Zirric Language Documentation',
  siteUrl: 'https://zirric.knabel.dev',

  // --- Branding ---
  logo: {
    light: '/assets/images/zirric.svg',
    dark: '/assets/images/zirric.svg',
    alt: 'Zirric Logo',
    href: '/',
  },
  favicon: '/assets/favicon.ico',
  customHead: [
    '<link rel="icon" type="image/svg+xml" href="/assets/images/favicon/favicon.svg">',
    '<link rel="icon" type="image/png" sizes="96x96" href="/assets/images/favicon/favicon-96x96.png">',
    '<link rel="apple-touch-icon" href="/assets/images/favicon/apple-touch-icon.png">',
    '<link rel="manifest" href="/assets/images/favicon/site.webmanifest">',
    '<link rel="icon" type="image/png" sizes="192x192" href="/assets/images/favicon/web-app-manifest-192x192.png">',
    '<link rel="icon" type="image/png" sizes="512x512" href="/assets/images/favicon/web-app-manifest-512x512.png">'
  ],

  // --- Source & Output ---
  srcDir: './',
  outputDir: '../site',
  customJs: [],

  // --- Theme & Layout ---
  theme: {
    name: 'ruby',           // Options: 'default', 'sky', 'ruby', 'retro'
    defaultMode: 'dark',    // 'light', 'dark', or 'system'
    enableModeToggle: true, // Show mode toggle button
    positionMode: 'top',    // 'top' or 'bottom'
    codeHighlight: true,    // Enable Highlight.js
    customCss: [],          // e.g. ['assets/css/custom.css']
  },

  // --- Features ---
  search: true,           // Built-in offline search
  minify: true,           // Minify HTML/CSS/JS in build
  autoTitleFromH1: true,  // Auto-generate page title from first H1
  copyCode: true,         // Show "copy" button on code blocks
  pageNavigation: true,   // Prev/Next buttons at bottom

  // --- Navigation (Sidebar) ---
  navigation: [
    { title: 'Overview', path: '/overview', icon: 'layout-list' },
    {
      title: 'Guides',
      path: '/guides',
      icon: 'book-open',
      collapsible: true,
      children: [
        { title: 'Getting Started', path: '/guides/getting-started', icon: 'rocket' },
        { title: 'Installation', path: '/guides/installation', icon: 'hard-drive' },
        { title: 'Styleguide', path: '/guides/styleguide', icon: 'paintbrush' },
      ]
    },
    {
      title: 'Tooling',
      path: '/tooling',
      icon: 'wrench',
      collapsible: true,
      children: [
        { title: 'Editor Support', path: '/tooling/editor-support', icon: 'dna' },
        { title: 'Package Manager', path: '/tooling/package-manager', icon: 'box' },
        { title: 'Compiler', path: '/tooling/compiler', icon: 'cpu' },
      ]
    },
    {
      title: 'Standard Library',
      path: '/stdlib',
      icon: 'library',
      collapsible: true,
      children: [
        { title: 'Prelude', path: '/stdlib/prelude', icon: 'sparkles' },
        {
          title: 'Future',
          path: '/stdlib/future',
          icon: 'flask-conical',
          collapsible: true,
          children: [
            { title: 'Prelude', path: '/stdlib/future/prelude', icon: 'sparkles' },
            { title: 'Reflect', path: '/stdlib/future/reflect', icon: 'scan' },
            { title: 'Cave', path: '/stdlib/future/cave', icon: 'package' },
            { title: 'Tasks', path: '/stdlib/future/tasks', icon: 'check-square' },
          ]
        },
      ]
    },
    {
      title: 'Proposals',
      path: '/proposals',
      icon: 'circle',
      collapsible: true,
      children: [
        // draft: circle
        // in-progress: circle-dot
        // implemented: circle-check
        // rejected: circle-x
        { icon: 'circle-dot', title: 'ZE-01 Language', path: '/proposals/ZE-001-base-language' },
        { icon: 'circle-dot', title: 'ZE-02 Cavefile', path: '/proposals/ZE-002-the-cavefile' },
        { icon: 'circle-x', title: 'ZE-03 Named Params', path: '/proposals/ZE-003-named-data-construction' },
        { icon: 'circle', title: 'ZE-04 Variadic', path: '/proposals/ZE-004-Variadic-Arguments' },
        { icon: 'circle', title: 'ZE-05 Mixins', path: '/proposals/ZE-005-Mixin-Type-Declarations' },
        { icon: 'circle', title: 'ZE-06 Parsing', path: '/proposals/ZE-006-Annotation-Based-Parsing-System' },
        { icon: 'circle', title: 'ZE-07 Binding', path: '/proposals/ZE-007-annotation-binding' },
        { icon: 'circle-dot', title: 'ZE-08 Errors', path: '/proposals/ZE-008-error-handling' },
        { icon: 'circle-dot', title: 'ZE-09 Option', path: '/proposals/ZE-009-option-values' },
        { icon: 'circle-dot', title: 'ZE-10 Iterable', path: '/proposals/ZE-010-iterable' },
        { icon: 'circle-dot', title: 'ZE-11 Zirric CLI', path: '/proposals/ZE-011-zirric-cli' },
        { icon: 'circle-dot', title: 'ZE-12 Type and Returns Sugar', path: '/proposals/ZE-012-type-and-returns-sugar' },
      ]
    },
    {
      title: 'Specification',
      icon: 'pencil-ruler',
      collapsible: true,
      children: [
        { title: 'Expressions', path: '/specification/expressions' },
        { title: 'Declarations', path: '/specification/declarations' },
        { title: 'Control Flow', path: '/specification/control-flow' },
        { title: 'Annotations', path: '/specification/annotations' },
        { title: 'Typesystem', path: '/specification/typesystem' },
      ]
    },
    { title: 'Repository', path: 'https://code.knabel.dev/zirric-lang/zirric', icon: 'git-graph', external: true },
    { title: 'Tree-Sitter-Zirric', path: 'https://code.knabel.dev/zirric-lang/tree-sitter-zirric', icon: 'git-branch', external: true },
  ],

  // --- Plugins ---
  plugins: {
    seo: {
      defaultDescription: 'Documentation built with docmd.',
      openGraph: {
        defaultImage: '/assets/images/logo@512x.png',
      },
    },
    sitemap: {
      defaultChangefreq: 'monthly', // e.g. 'daily', 'weekly', 'monthly'
      defaultPriority: 0.8          // Priority between 0.0 and 1.0
    }
  },

  // --- Footer ---
  footer: '© ' + new Date().getFullYear() + ' Zirric. Built with [docmd](https://docmd.io).',
  
  // --- Edit Link ---
  editLink: {
    enabled: false,
    baseUrl: 'https://code.knabel.dev/zirric-lang/zirric/edit/main/docs',
    text: 'Edit this page'
  }
};
