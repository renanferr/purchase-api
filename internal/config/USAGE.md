// Example Usage of Go Options Pattern

// The application now uses the idiomatic Go options pattern for configuration.
// This provides maximum flexibility and composability.

// BASIC USAGE (uses environment variables and defaults):
// cfg, err := config.BuildConfig(ctx)
// if err != nil {
//   log.Fatal(err)
// }

// WITH CUSTOM OPTIONS:
// cfg, err := config.BuildConfig(ctx,
//   config.WithAPIPort("9000"),
//   config.WithRealProvider(),
//   config.WithHTTPTimeout(30 * time.Second),
// )

// SAMPLE PROVIDER OPTIONS:
// Option 1: Auto-detect from TREASURY_PROVIDER env var (default: "sample")
// cfg, _ := config.BuildConfig(ctx)

// Option 2: Explicitly use sample provider with custom test data
// cfg, _ := config.BuildConfig(ctx,
//   config.WithSampleProvider(
//     decimal.NewFromFloat(0.85),
//     "GBP",
//     time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
//   ),
// )

// Option 3: Use sample provider with sensible defaults
// cfg, _ := config.BuildConfig(ctx,
//   config.WithSampleProviderDefaults(),
// )

// Option 4: Use real Treasury API
// cfg, _ := config.BuildConfig(ctx,
//   config.WithRealProvider(),
// )

// ENVIRONMENT VARIABLE DETECTION:
// - TREASURY_PROVIDER: "sample" (default) or "real"
// - DATABASE_URL: PostgreSQL connection string
// - API_PORT: HTTP server port (default: 8080)
// - LOG_LEVEL: DEBUG, INFO (default), WARN, ERROR

// OPTIONS COMPOSITION:
// Multiple options can be combined. Later options override earlier ones.
// cfg, _ := config.BuildConfig(ctx,
//   config.WithAPIPort("9000"),
//   config.WithAPIPort("8080"),  // This will win
// )
