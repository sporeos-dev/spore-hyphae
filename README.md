# spore-hyphae
An extension of the hub to interact securely with the user space.

## OS-authenticated confirmation

`HYPHAE.confirm` accepts a requesting node ID and command, then returns exactly
one `yes` or `no` response flag. On macOS it uses system authentication with
Touch ID and signed-in-user password fallback. The system prompt identifies the
node and command, but does not display raw command arguments, which can contain
personal or secret values.

Hyphae does not authorize or commit an operation. The hub decides when a
protected operation requires confirmation, binds the response to its pending
transaction, and performs any `_spore`-owned state change. Linux and Windows
currently return an explicit unsupported-platform error until native
authentication providers are implemented.
