module github.com/pawsql/dbx-plugin-pawsql

go 1.22

require github.com/t8y2/dbx/plugins/sdk/go/dbx-plugin-sdk v0.1.0

// The DBX Go SDK is a nested module in the t8y2/dbx repo with no published
// version tag, so it cannot be fetched from a module proxy. It is vendored
// under third_party/ and wired up with a local replace (the same mechanism the
// plugin CLI injects via DBX_PLUGIN_SDK_ROOT).
replace github.com/t8y2/dbx/plugins/sdk/go/dbx-plugin-sdk => ./third_party/dbx-plugin-sdk
