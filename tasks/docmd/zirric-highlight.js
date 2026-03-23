const hljs = require("highlight.js");
const zirricLanguage = require("./highlight-zirric.js");

hljs.registerLanguage("zirric", zirricLanguage);
hljs.registerLanguage("zirr", zirricLanguage);

module.exports = {};
