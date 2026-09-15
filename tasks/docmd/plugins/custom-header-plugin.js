module.exports = {
  plugin: {
    "name": "custom-header",
    "version": "1.0.0",
    "capabilities": ["head"]
  },
  generateMetaTags: cfg => {
    return cfg.plugins["./plugins/custom-header-plugin.js"].metaTags.join('\n');
  },
  generateScripts: cfg => {
    return {
      headScriptsHtml: cfg.plugins["./plugins/custom-header-plugin.js"].headScriptsHtml.join('\n'),
    }
  },
};
