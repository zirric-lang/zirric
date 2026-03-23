/**
 * highlight.js language definition for Zirric.
 * @param {Object} hljs - The highlight.js instance
 * @returns {Object} Language definition
 */
module.exports = function (hljs) {
  const KEYWORDS = {
    keyword: [
      "fn", "return",
      "if", "else",
      "for", "break", "continue",
      "switch", "case", "is",
      "extern", "type",
      "mod", "import",
    ],
    "keyword.declaration": [
      "const", "var", "data", "union", "attr",
    ],
    literal: ["true", "false"],
  };

  const COMMENT = hljs.COMMENT("//", "$");
  const BLOCK_COMMENT = hljs.COMMENT("/\\*", "\\*/");

  const ESCAPE = {
    scope: "char.escape",
    begin: /\\(?:[nrtf"\\]|\d{2,3}|x[0-9a-fA-F]{2,}|u[0-9a-fA-F]{4}|U[0-9a-fA-F]{8})/,
  };

  const STRING = {
    scope: "string",
    begin: '"',
    end: '"',
    contains: [ESCAPE],
  };

  const NUMBER = {
    scope: "number",
    // floats before ints to avoid partial matches
    variants: [
      { begin: /-?(?:(?:0|[1-9]\d*)(?:\.\d+)|\.\d+)(?:[eE][+\-]?\d+)?/ },
      { begin: /-?[0-9]+/ },
    ],
  };

  // @Attribute or @namespace.Attribute
  const ATTRIBUTE = {
    scope: "meta",
    begin: /@[a-zA-Z_\u00C0-\uFFFF][a-zA-Z0-9_.\u00C0-\uFFFF]*/,
  };

  // Type names: identifiers after `:` or `->` and in `is` patterns
  const TYPE_HINT = {
    scope: "type",
    begin: /(?<=(?:->\s*|:\s*|is\s+))[A-Z][a-zA-Z0-9_]*/,
  };

  const OPERATOR = {
    className: 'operator',
    begin: /\+\+|--|==|!=|>=|<=|&&|\|\|=>|<-|[=+\-*/<>!]/,
    relevance: 0,
  };

  return {
    name: "Zirric",
    aliases: ["zirr"],
    case_insensitive: false,
    keywords: KEYWORDS,
    contains: [
      COMMENT,
      BLOCK_COMMENT,
      STRING,
      NUMBER,
      ATTRIBUTE,
      TYPE_HINT,
      OPERATOR,
      {
        // fn declarations and calls: identifier followed by `(`
        scope: "title.function",
        begin: /\b(?!(?:fn|if|else|for|switch|case|return|break|continue)\b)[a-zA-Z_\u00C0-\uFFFF][a-zA-Z0-9_\u00C0-\uFFFF]*(?=\s*\()/,
      },
    ],
  };
};
