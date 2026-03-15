module.exports = {
  injectHead: cfg => {
    return cfg.plugins["custom-header"].header.join('\n');
  },
  generateMetaTags: cfg => {
    return cfg.plugins["custom-header"].header.join('\n');
  },
};
