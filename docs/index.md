---
title: Zirric
description: A compact, expression-first language with annotations and a small runtime.
noStyle: true
components:
  meta: true
  favicon: true
  scripts: false
---

<div class="landing">
  <nav class="top-nav">
    <a class="brand" href="/overview">
      <img src="/assets/images/zirric.svg" alt="Zirric logo">
    </a>
    <div class="nav-links">
      <a href="/guides/getting-started">Get Started</a>
      <a href="/overview">Docs</a>
      <a href="/proposals">Proposals</a>
    </div>
  </nav>

  <header class="hero">
    <div class="hero-content">
      <p class="eyebrow">Zirric Language</p>
      <h1>Explicit structure. Expression-first code.</h1>
      <p class="lead">
        Zirric is a compact language that favors declarations, readable data
        modeling, and annotations that describe capabilities without interfaces.
      </p>
      <div class="hero-actions">
        <a class="button primary" href="/guides/getting-started">Get Started</a>
        <a class="button" href="/overview">Read the Docs</a>
        <a class="button ghost" href="/proposals">Browse Proposals</a>
      </div>
      <p class="note">Experimental: features and syntax may change.</p>
    </div>
    <div class="hero-code">
      <pre><code class="language-zirric">annotation Countable {
    @Returns(Int)
    length(@Has(Countable) value)
}

@Countable({ v -> v.length })
data Bag {
    items
    length
}

@Returns(Result)
func summarize(@Bag bag) {
    let length = Countable(bag).length(bag)
    return if length > 0 {
        Ok(length)
    } else {
        Err("empty")
    }
}
</code></pre>
    </div>
  </header>

  <section class="cards">
    <article class="card">
      <h3>Declarations first</h3>
      <p>Small set of primitives: <code>let</code>, <code>func</code>, <code>data</code>, <code>enum</code>, <code>annotation</code>, <code>module</code>.</p>
    </article>
    <article class="card">
      <h3>Expression-oriented</h3>
      <p><code>if</code> and <code>for</code> return values when you need them, so data flow stays explicit.</p>
    </article>
    <article class="card">
      <h3>Annotations over interfaces</h3>
      <p>Capabilities are declared, composed, and exposed to tooling.</p>
    </article>
    <article class="card">
      <h3>Readable modeling</h3>
      <p>Records and tagged unions stay close to the domain and are easy to reason about.</p>
    </article>
  </section>

  <section class="quick-links">
    <h2>Explore the docs</h2>
    <div class="link-grid">
      <a class="link-card" href="/guides/getting-started">
        <h3>Getting Started</h3>
        <p>A guided tour of the language.</p>
      </a>
      <a class="link-card" href="/syntax/expressions">
        <h3>Syntax Reference</h3>
        <p>The precise language surface.</p>
      </a>
      <a class="link-card" href="/tooling">
        <h3>Tooling</h3>
        <p>Syntax highlighting and compiler notes.</p>
      </a>
      <a class="link-card" href="/guides/styleguide">
        <h3>Styleguide</h3>
        <p>Conventions for clear Zirric code.</p>
      </a>
    </div>
  </section>
</div>

<style>
.landing {
  margin: 0 auto;
  padding: 1.25rem 2.5rem 3.5rem;
  max-width: 1180px;
  color: var(--text);
  --text: #f8f2f2;
  --muted: rgba(255, 255, 255, 0.75);
  --muted-2: rgba(255, 255, 255, 0.6);
  --border: rgba(255, 255, 255, 0.12);
  --surface: rgba(255, 255, 255, 0.06);
  --surface-strong: rgba(10, 10, 10, 0.75);
  --hero-grad-a: rgba(235, 90, 90, 0.15);
  --hero-grad-b: rgba(255, 200, 160, 0.12);
  --code-text: #f7efef;
}
.landing,
.landing * {
  box-sizing: border-box;
}
[data-theme="light"] .landing {
  --text: #2a1c1c;
  --muted: rgba(42, 28, 28, 0.7);
  --muted-2: rgba(42, 28, 28, 0.55);
  --border: rgba(42, 28, 28, 0.15);
  --surface: rgba(0, 0, 0, 0.03);
  --surface-strong: rgba(255, 255, 255, 0.9);
  --hero-grad-a: rgba(255, 120, 120, 0.2);
  --hero-grad-b: rgba(255, 205, 160, 0.32);
  --code-text: #2a1c1c;
}
.top-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 2rem;
  padding: 0 0 1.25rem;
  flex-wrap: wrap;
  min-width: 0;
}
.top-nav .brand {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--text);
  text-decoration: none;
}
.top-nav .brand img {
  height: 40px;
  width: auto;
}
.nav-links {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
  min-width: 0;
}
.nav-links a {
  text-transform: uppercase;
  letter-spacing: 0.18em;
  color: var(--muted);
  text-decoration: none;
  font-weight: 600;
}
.nav-links a:hover {
  color: var(--text);
}
.hero {
  display: grid;
  gap: 2rem;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.9fr);
  align-items: center;
  padding: 2.5rem 2.5rem;
  border-radius: 1.5rem;
  background: linear-gradient(135deg, var(--hero-grad-a), var(--hero-grad-b));
  border: 1px solid var(--border);
}
.hero-content,
.hero-code {
  min-width: 0;
}
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  font-size: 0.75rem;
  color: var(--muted);
  margin: 0 0 0.5rem;
}
.hero h1 {
  font-size: clamp(2.2rem, 4vw, 3.4rem);
  margin: 0 0 1rem;
  line-height: 1.1;
}
.lead {
  font-size: 1.1rem;
  color: var(--muted);
  margin: 0 0 1.5rem;
}
.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}
.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.65rem 1.2rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  color: var(--text);
  text-decoration: none;
  font-weight: 600;
  background: var(--surface);
}
.button.primary {
  background: linear-gradient(135deg, #ff6b6b, #ff9f68);
  border: none;
  color: #1a0d0d;
}
.button.ghost {
  border-color: var(--border);
  background: transparent;
}
.note {
  margin-top: 1rem;
  font-size: 0.85rem;
  color: var(--muted-2);
}
.hero-code pre {
  margin: 0;
  border-radius: 1rem;
  background: var(--surface-strong);
  border: 1px solid var(--border);
  padding: 1.25rem;
  font-size: 0.9rem;
  color: var(--code-text);
  max-width: 100%;
  width: 100%;
  overflow-x: auto;
}
.cards {
  margin-top: 2.5rem;
  display: grid;
  gap: 1.5rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}
.card {
  padding: 1.5rem;
  border-radius: 1.2rem;
  background: var(--surface);
  border: 1px solid var(--border);
}
.card h3 {
  margin-top: 0;
}
.quick-links {
  margin-top: 3rem;
}
.link-grid {
  display: grid;
  gap: 1.25rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}
.link-card {
  display: block;
  padding: 1.4rem;
  border-radius: 1rem;
  background: var(--surface);
  border: 1px solid var(--border);
  color: inherit;
  text-decoration: none;
}
.link-card h3 {
  margin: 0 0 0.5rem;
}
@media (max-width: 960px) {
  .hero {
    grid-template-columns: 1fr;
  }
  .hero-code {
    order: 2;
  }
  .top-nav {
    flex-direction: column;
    align-items: flex-start;
  }
  .top-nav .brand {
    align-self: center;
  }
  .nav-links {
    align-self: center;
    justify-content: center;
  }
}
@media (max-width: 640px) {
  .landing {
    padding: 1rem 1.25rem 2.5rem;
  }
  .hero {
    margin-left: -1.25rem;
    margin-right: -1.25rem;
    padding: 2rem 1.25rem;
    border: 0;
    border-radius: 0;
  }
  .nav-links {
    width: 100%;
    gap: 0.6rem 1rem;
    justify-content: center;
  }
  .nav-links a {
    font-size: 0.75rem;
    letter-spacing: 0.12em;
  }
}
</style>
