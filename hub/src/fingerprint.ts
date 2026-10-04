export async function visitorId(): Promise<string> {
  const { default: FingerprintJS } = await import('@fingerprintjs/fingerprintjs')
  const agent = await FingerprintJS.load()
  const result = await agent.get()
  return result.visitorId
}
