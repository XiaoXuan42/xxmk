# gomk

**gomk** is a markdown parsing library written in go. It mainly aims to align with the basic-syntax listed in [markdown-guide](https://www.markdownguide.org/basic-syntax/). 

However, **gomk** is not primarily created to convert markdown to html, but to help analyze the markdown-like file's structure and content. So **gomk** pays special attention to extensibility and focuses less on how to style correctly.

Any difference and extend syntax are listed below.

## Difference
- **gomk** by default only supports **block level** parsing, which means syntax like emphasis and italic won't be specially treated, however you can add the inline extension to enable this.
- Don't support alternative header syntax:
```
Header
===
```
- Don't support multi-line blockquotes.
- Don't support nested blockquotes.
- Don't support list.
- Don't support mixture of \*/\+/\- in list:
```
* item1
+ item2
```
- Don't support double backticks to escape backtickes:
```
``a `b` c``
```
- Don't support horizontal rules.
- Don't support reference-style links.
- Don't support linked images.
- Don't support code blocks by indentation:
```
    line1
    line2
```

## Extended syntax
- Tables.
- Three backticks to trigger fenced code blocks.

## Roadmap
- [ ] multi-line quoteblock
- [ ] nested quoteblock
- [ ] list
- [ ] double backticks to escape backtices
- [ ] linked images
- [ ] reference style links
- [ ] nested block by identation
- [ ] reference-style links
- [ ] horizontal rules
- [ ] strikethrough
- [ ] html tags
- [ ] footnotes (as extension)
- [ ] heading ids (as extension)
- [ ] definition list (as extension)
- [ ] task lists (as extension)
- [ ] subscript/superscript (as extension)

## Won't included forever
- Alternative header syntax
- Mixture of \*/\+/\- in list
- Highlighting
