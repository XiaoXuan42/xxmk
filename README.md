# gomk

**gomk** is a markdown parsing library written in go. It mainly aims to align with the basic-syntax listed in [markdown-guide](https://www.markdownguide.org/basic-syntax/) and some additional extended syntaxes listed in [extend](https://www.markdownguide.org/extended-syntax/). 

However, **gomk** is not primarily created to convert markdown to html, but to help analyze the markdown-like file's structure and content. So **gomk** pays special attention to extensibility and focuses less on how to style correctly.

Some cautions are listed below.

## Cautions
- **gomk** by default only supports **block level** parsing, which means syntax like emphasis and italic won't be specially treated, however you can add the inline extension to enable this.
- Don't support alternative header syntax:
```
Header
===
```
- Multi-line quote block: multi-line quote block will be treated as multiple ast nodes, each corresponds to one line.
- Don't support mixture of \*/\+/\- in list:
```
* item1
+ item2
```
- Don't support highlight: you can use `<mark> </mark>` instead.
```
==highlight==
```
- Don't support definition list:
```
Term
: definition1
: definition2
```
- Don't support subscript/superscript. Put them in math block instead:
```
x_1^2
```
- Url/email address match may be wrong for links due to the wrong regexpr.
- Html tag is parsed to a token, there won't be a DOM tree in the final AST.
- Unordered list must start with `-` symbol.
- Ordered list starts from the index given in the first line of the list.

## Roadmap
- [ ] Nested block by identation
- [ ] Heading Ids(as extension)