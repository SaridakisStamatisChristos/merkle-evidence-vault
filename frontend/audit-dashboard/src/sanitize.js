import createDOMPurify from 'dompurify'

// sanitizeHTML cleans HTML strings before insertion into the DOM.
// Use `createDOMPurify(window)` to bind to the provided window (jsdom or real).
export function sanitizeHTML(windowLike, html) {
  if (windowLike && windowLike.document) {
    const purify = createDOMPurify(windowLike)
    return purify.sanitize(html)
  }
  // fallback: if running in browser where global window exists
  if (typeof window !== 'undefined' && window && (window.document)) {
    const purify = createDOMPurify(window)
    return purify.sanitize(html)
  }
  // if no DOM available, defensively escape angle brackets
  return html.replaceAll('<', '&lt;').replaceAll('>', '&gt;')
}

export default sanitizeHTML
