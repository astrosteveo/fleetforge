# Simple documentation PDF generation
# Requires pandoc and a LaTeX engine installed (e.g., tectonic or xelatex)

DOCS_MD := $(wildcard docs/*.md)
DOCS_PDF := $(DOCS_MD:.md=.pdf)
PANDOC_ENGINE ?= tectonic
PANDOC_FROM ?= gfm+footnotes+autolink_bare_uris

all: pdfs

pdfs: $(DOCS_PDF)
	@echo "Generated: $(DOCS_PDF)"

# Pattern rule: docs/file.md -> docs/file.pdf
# Uses --metadata to set a basic title, and table of contents.
# To force rebuild: make clean && make

docs/%.pdf: docs/%.md
	@echo "Converting $< -> $@"
	@pandoc "$<" \
	  --from $(PANDOC_FROM) \
	  --pdf-engine=$(PANDOC_ENGINE) \
	  --toc --toc-depth=3 \
	  -V geometry:margin=1in \
	  -V linkcolor:blue \
	  -o "$@"

clean:
	@rm -f $(DOCS_PDF)
	@echo "Removed generated PDFs"

.PHONY: all pdfs clean
