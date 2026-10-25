# spore-hyphae
An extension of the hub to interact securely with the user space.

## OS-authenticated confirmation

`HYPHAE.confirm` accepts a requesting node ID and command, then returns exactly
one `yes` or `no` response flag. On macOS it uses system authentication with
Touch ID and signed-in-user password fallback. On Linux it requests
authorization from Polkit, which delegates authentication to the user's
registered OS authentication agent. Hyphae never receives authentication
credentials. Since Hyphae is an unprivileged caller, Polkit does not allow it
to supply dynamic prompt details; the installed action uses a fixed message.
The Linux installer registers the required Polkit action. Linux installations
need a Polkit authentication agent in the user's session to approve requests.

Hyphae does not authorize or commit an operation. The hub decides when a
protected operation requires confirmation, binds the response to its pending
transaction, and performs any `_spore`-owned state change. Windows currently
returns an explicit unsupported-platform error until a native authentication
provider is implemented.
