function fallbackCopyToClipboard(text: string): boolean {
  const textArea = document.createElement('textarea')
  textArea.value = text

  textArea.style.position = 'fixed'
  textArea.style.left = '-999999px'
  textArea.style.top = '-999999px'
  textArea.style.opacity = '0'
  textArea.setAttribute('readonly', '')

  document.body.appendChild(textArea)

  try {
    textArea.focus()
    textArea.select()

    const range = document.createRange()
    range.selectNodeContents(textArea)
    const selection = window.getSelection()
    if (selection) {
      selection.removeAllRanges()
      selection.addRange(range)
    }
    textArea.setSelectionRange(0, text.length)

    const successful = document.execCommand('copy')
    document.body.removeChild(textArea)
    const selectionAfter = window.getSelection()
    if (selectionAfter) {
      selectionAfter.removeAllRanges()
    }

    return successful
  } catch (err) {
    console.error('Fallback copy failed:', err)
    document.body.removeChild(textArea)
    return false
  }
}

export async function copyToClipboard(text: string): Promise<boolean> {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return false
  }

  if (navigator?.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (error) {
      console.warn('Clipboard API failed, trying fallback method:', error)
      return fallbackCopyToClipboard(text)
    }
  } else {
    return fallbackCopyToClipboard(text)
  }
}
