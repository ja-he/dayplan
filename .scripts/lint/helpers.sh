# filters for and formats the stylesheet identifiers as
#     `<Go identifier> <YAML identifier>`
enumerate_stylesheet_identifiers() {
  cat src/config/config.go \
    | sed -n '/type Stylesheet struct/,/}/p' \
    | grep 'Styling' \
    | sed 's/^\s*\([A-Z][a-zA-Z]\+\)\s\+Styling\s\+`yaml:"\([a-z-]*\)"`\s*$/\1 \2/'
}
