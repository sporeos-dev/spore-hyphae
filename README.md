# spore-hyphae
An extension of the hub to interact securely with the user space.

## OS-authenticated confirmation

`HYPHAE.confirm` accepts a requesting node ID and command, then returns exactly
one `yes` or `no` response flag. On macOS it uses system authentication with
Touch ID and signed-in-user password fallback. On Windows it uses the native
Windows Forms UI to display the requesting node and command, then requires the
user to type the exact text `yes` before enabling approval. Closing the dialog,
cancelling, entering any other text, or a dialog error returns `no`. This is a
deliberate user-review checkpoint; unlike the macOS and Linux paths, it is not
OS-authenticated and does not defend against a malicious process with
UI-automation capability. The prompt does not display raw command arguments,
which can contain personal or secret values.

On Linux it requests authorization from Polkit, which delegates authentication
to the user's
registered OS authentication agent. Hyphae never receives authentication
credentials. Since Hyphae is an unprivileged caller, Polkit does not allow it
to supply dynamic prompt details; the installed action uses a fixed message.
The Linux installer registers the required Polkit action. Linux installations
need a Polkit authentication agent in the user's session to approve requests.
