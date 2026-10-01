# Authentication in mcpd

The experimental transport packages support local, GitHub, and custom providers
through `transport.AuthConfig` and `transport.SetupAuthentication`.

The command accepts `-oauth-provider`, defaulting to `local`, but does not wire
authentication into its listeners. It has no flags for enabling authentication,
setting OAuth credentials, or loading local users. Selecting a provider does not
protect an endpoint. Applications using the transport packages must configure and
attach authentication themselves.
