# Performance Tuning Example

This example demonstrates how to configure the Reddit client library for optimal performance in different scenarios. It shows the impact of various transport configurations and compression settings on throughput and resource usage.

## What This Example Demonstrates

### Configuration Types

1. **Default Configuration**
   - Balanced settings suitable for most applications
   - Compression enabled for bandwidth efficiency
   - Moderate connection pooling

2. **No Compression Configuration**
   - Shows performance impact when compression is disabled
   - Useful for understanding bandwidth savings from gzip compression
   - Generally not recommended except for debugging

3. **Low-Throughput Configuration**
   - Conservative settings for resource-constrained environments
   - Minimal connection pooling and lower rate limits
   - Ideal for mobile apps, serverless functions, or low-traffic applications

4. **High-Throughput Configuration**
   - Optimized for maximum performance and concurrent requests
   - High connection limits and aggressive connection reuse
   - Best for data analysis, bulk operations, or high-traffic servers

### Performance Metrics

The example measures:

- Total execution time for concurrent requests
- Success rate across multiple subreddits
- Average time per request
- Error rates and types

## Running the Example

### Prerequisites

1. **Environment Variables**: Create a `.env` file in this directory with your Reddit API credentials:

   ```env
   REDDIT_CLIENT_ID=your_client_id_here
   REDDIT_CLIENT_SECRET=your_client_secret_here
   ```

2. **Dependencies**: The example uses:
   - `github.com/joho/godotenv` for environment variable loading
   - Standard library packages for HTTP transport configuration

### Execution

```bash
# From the performance-tuning directory
go run main.go

# Or build and run
go build -o performance-tuning main.go
./performance-tuning
```

### Expected Output

The example will test each configuration sequentially, showing:

- Real-time progress as it fetches posts from multiple subreddits
- Performance metrics for each configuration
- Detailed recommendations for different use cases

## Key Configuration Parameters

### Transport Settings

- **MaxIdleConns**: Total number of idle connections across all hosts
- **MaxIdleConnsPerHost**: Maximum idle connections per Reddit endpoint
- **IdleConnTimeout**: How long to keep idle connections open
- **MaxConnsPerHost**: Total connections per host (0 = unlimited)
- **DisableKeepAlives**: Whether to reuse TCP connections

### Client Settings

- **Rate Limiting**: Requests per minute and burst capacity
- **Compression**: Gzip compression for response bodies
- **Timeouts**: Request timeout values
- **User Agent**: Custom user agent strings

## Performance Recommendations

### For Most Applications (Default)

```go
// Default settings are usually optimal
client, err := reddit.NewClient(auth)
```

### For High-Volume Applications

```go
highThroughputConfig := &reddit.TransportConfig{
    MaxIdleConns:        200,
    MaxIdleConnsPerHost: 20,
    IdleConnTimeout:     120 * time.Second,
    MaxConnsPerHost:     0, // No limit
}

client, err := reddit.NewClient(auth,
    reddit.WithTransportConfig(highThroughputConfig),
    reddit.WithRateLimit(100, 10),
)
```

### For Resource-Constrained Environments

```go
lowThroughputConfig := &reddit.TransportConfig{
    MaxIdleConns:        10,
    MaxIdleConnsPerHost: 2,
    IdleConnTimeout:     30 * time.Second,
    MaxConnsPerHost:     5,
}

client, err := reddit.NewClient(auth,
    reddit.WithTransportConfig(lowThroughputConfig),
    reddit.WithRateLimit(30, 3),
)
```

## Understanding the Results

### Compression Benefits

- **Bandwidth Savings**: 30-60% reduction in response size
- **Transfer Speed**: Faster downloads, especially on slower connections
- **Cost Reduction**: Lower bandwidth usage and data transfer costs

### Connection Pooling Impact

- **Higher Limits**: Better performance for concurrent requests
- **Lower Limits**: Reduced memory usage and connection overhead
- **Keep-Alives**: Significant performance improvement through connection reuse

### Rate Limiting Considerations

- **Conservative Limits**: Better compliance with Reddit's API policies
- **Aggressive Limits**: Higher throughput but risk of rate limiting
- **Burst Capacity**: Handles temporary spikes in request volume

## Monitoring and Optimization

### Key Metrics to Watch

1. **Response Times**: Average and 95th percentile latencies
2. **Error Rates**: HTTP errors, timeouts, and rate limiting
3. **Memory Usage**: Connection pool and response buffer sizes
4. **CPU Usage**: Compression and JSON parsing overhead

### Optimization Tips

1. **Start with defaults** and measure actual performance
2. **Increase connection limits** if you make many concurrent requests
3. **Keep compression enabled** unless debugging HTTP traffic
4. **Monitor rate limiting** headers and adjust limits accordingly
5. **Use circuit breakers** for resilient high-throughput applications

## Integration with Monitoring

This example can be extended to integrate with monitoring systems:

```go
// Add request interceptor for metrics collection
client, err := reddit.NewClient(auth,
    reddit.WithRequestInterceptor(func(req *http.Request) {
        // Record request start time, add tracing headers
    }),
    reddit.WithResponseInterceptor(func(resp *http.Response) {
        // Record response time, status codes, errors
    }),
)
```

## Related Examples

- **[Basic Example](../basic/)**: Simple client usage
- **[Comprehensive Example](../comprehensive/)**: Full feature demonstration
- **[Interceptors Example](../interceptors/)**: Request/response middleware patterns

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Reduce `MaxIdleConns` and `MaxIdleConnsPerHost`
2. **Slow Performance**: Increase connection limits and enable compression
3. **Rate Limiting**: Reduce `WithRateLimit` values or implement backoff
4. **Connection Errors**: Check network settings and proxy configuration

### Debug Mode

To debug HTTP traffic, temporarily disable compression:

```go
client, err := reddit.NewClient(auth,
    reddit.WithNoCompression(), // Only for debugging
)
```

This allows you to see raw HTTP responses but significantly increases bandwidth usage.
