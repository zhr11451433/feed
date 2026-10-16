export function reportError(error: Error, context?: Record<string, unknown>) {
  if (import.meta.env.DEV) {
    console.error('[ErrorReporter]', error.message, context)
  }
}