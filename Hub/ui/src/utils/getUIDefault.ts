// Extracts a displayable default from a sensor manifest env-var default.
// Manifest defaults may contain Go template expressions (e.g. `{{ else }}...{{ end }}`
// or `{{ funcName 8080 }}`); this resolves them to a plain preview string.
export const getUIDefault = (def: any): string => {
  if (def === undefined || def === null) return ''
  const strDef = String(def)
  if (!strDef.includes('{{')) return strDef
  const elseMatch = strDef.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/)
  if (elseMatch) return elseMatch[1].trim()
  const funcMatch = strDef.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/)
  if (funcMatch) return funcMatch[1].trim()
  return ''
}
