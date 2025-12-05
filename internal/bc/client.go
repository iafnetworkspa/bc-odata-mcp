package bc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ODataResponse represents a paginated OData response
type ODataResponse struct {
	Value    []map[string]interface{} `json:"value"`
	NextLink string                   `json:"@odata.nextLink,omitempty"`
}

// Client handles HTTP requests to Business Central API
type Client struct {
	config     Config
	auth       *Auth
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Business Central API client
func NewClient(cfg Config, auth *Auth) *Client {
	timeout := cfg.APITimeout
	if timeout == 0 {
		timeout = 90
	}
	return &Client{
		config: cfg,
		auth:   auth,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		baseURL: cfg.BasePath,
	}
}

// Get makes a GET request to the Business Central API with automatic token handling
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.GetWithRetry(ctx, endpoint, 5)
}

// GetWithRetry makes a GET request with retry logic
func (c *Client) GetWithRetry(ctx context.Context, endpoint string, maxRetries int) (*http.Response, error) {
	log := log.With().
		Str("component", "bc_client").
		Str("endpoint", endpoint).
		Int("max_retries", maxRetries).
		Logger()

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff for non-rate-limit errors: 2s, 4s, 8s
			backoff := time.Duration(1<<uint(attempt-1)) * 2 * time.Second
			log.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Err(lastErr).
				Msg("Retrying API request after error")

			select {
			case <-ctx.Done():
				log.Error().Err(ctx.Err()).Msg("Context cancelled during retry")
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		log.Debug().
			Int("attempt", attempt+1).
			Msg("Getting OAuth token")

		token, err := c.auth.GetToken()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get OAuth token")
			return nil, fmt.Errorf("failed to get token: %w", err)
		}

		// Construct full URL
		fullURL := c.baseURL + endpoint

		// Parse URL to ensure proper encoding
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			log.Error().Err(err).Str("url", fullURL).Msg("Failed to parse URL")
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}

		fullURL = parsedURL.String()

		log.Debug().
			Str("url", fullURL).
			Str("endpoint", endpoint).
			Str("base_url", c.baseURL).
			Msg("Creating HTTP request")

		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			log.Error().Err(err).Str("url", fullURL).Msg("Failed to create HTTP request")
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		log.Debug().Msg("Sending HTTP request")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Warn().Err(err).Msg("HTTP request failed")
			lastErr = err
			continue
		}

		log.Debug().Int("status_code", resp.StatusCode).Msg("Received HTTP response")

		// Check for unauthorized (401) - token may have expired, refresh and retry
		if resp.StatusCode == http.StatusUnauthorized {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			log.Warn().
				Int("status_code", resp.StatusCode).
				Str("status", resp.Status).
				Str("response_body", string(bodyBytes)).
				Msg("Unauthorized (401) - token may have expired, refreshing token")

			// Invalidate current token
			c.auth.InvalidateToken()

			// Refresh token and retry
			newToken, err := c.auth.GetToken()
			if err != nil {
				log.Error().Err(err).Msg("Failed to refresh token after 401")
				lastErr = fmt.Errorf("failed to refresh token: %w", err)
				continue
			}

			log.Info().Msg("Token refreshed successfully, retrying request")

			// Update token in request and retry
			req.Header.Set("Authorization", "Bearer "+newToken)

			// Retry the request with new token
			resp, err = c.httpClient.Do(req)
			if err != nil {
				log.Warn().Err(err).Msg("HTTP request failed after token refresh")
				lastErr = err
				continue
			}

			log.Debug().Int("status_code", resp.StatusCode).Msg("Received HTTP response after token refresh")
		}

		// Check for rate limiting (429) - needs special handling
		if resp.StatusCode == http.StatusTooManyRequests {
			// Try to read Retry-After header
			retryAfter := resp.Header.Get("Retry-After")
			var backoffDuration time.Duration

			if retryAfter != "" {
				// Parse Retry-After header (can be seconds or HTTP date)
				if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
					backoffDuration = seconds
				} else {
					// Try parsing as integer seconds
					if secs, err := strconv.ParseInt(retryAfter, 10, 64); err == nil {
						backoffDuration = time.Duration(secs) * time.Second
					} else {
						// Fallback to exponential backoff
						backoffDuration = time.Duration(attempt+1) * 5 * time.Second
					}
				}
			} else {
				// No Retry-After header, use exponential backoff: 5s, 10s, 20s
				backoffDuration = time.Duration(1<<uint(attempt)) * 5 * time.Second
			}

			log.Warn().
				Int("status_code", resp.StatusCode).
				Str("status", resp.Status).
				Str("retry_after", retryAfter).
				Dur("backoff", backoffDuration).
				Int("attempt", attempt+1).
				Msg("Rate limit exceeded (429), waiting before retry")

			resp.Body.Close()
			lastErr = fmt.Errorf("rate limit exceeded (429)")

			// Wait before retrying
			select {
			case <-ctx.Done():
				log.Error().Err(ctx.Err()).Msg("Context cancelled during rate limit wait")
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
			}
			continue
		}

		// Check for other server errors (5xx)
		if resp.StatusCode >= 500 {
			log.Warn().
				Int("status_code", resp.StatusCode).
				Str("status", resp.Status).
				Msg("Server error, will retry")
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Debug().Int("status_code", resp.StatusCode).Msg("Request successful")
			return resp, nil
		}

		// Client error (4xx) - read body for error details
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		log.Error().
			Int("status_code", resp.StatusCode).
			Str("status", resp.Status).
			Str("response_body", string(bodyBytes)).
			Str("url", fullURL).
			Msg("Client error (4xx), not retrying")
		return nil, fmt.Errorf("client error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	log.Error().
		Int("attempts", maxRetries).
		Err(lastErr).
		Msg("Max retries exceeded")
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// GetPaginated fetches all pages of an OData endpoint
func (c *Client) GetPaginated(ctx context.Context, endpoint string) ([]map[string]interface{}, error) {
	log := log.With().
		Str("component", "bc_client").
		Str("endpoint", endpoint).
		Logger()

	log.Info().Msg("Fetching paginated data from Business Central API")

	var allResults []map[string]interface{}
	currentEndpoint := endpoint
	skipCount := 0
	pageNum := 1

	// Check if $top is specified in the endpoint
	maxResults := -1 // -1 means no limit
	if strings.Contains(endpoint, "$top=") {
		// Extract $top value from endpoint
		topIndex := strings.Index(endpoint, "$top=")
		if topIndex != -1 {
			topPart := endpoint[topIndex+5:]
			// Find where $top value ends (either & or end of string)
			endIndex := strings.Index(topPart, "&")
			if endIndex == -1 {
				endIndex = len(topPart)
			}
			topValue := strings.TrimSpace(topPart[:endIndex])
			if top, err := strconv.Atoi(topValue); err == nil {
				maxResults = top
				log.Debug().Int("max_results", maxResults).Msg("Found $top parameter, limiting results")
			}
		}
	}

	// Rate limiting: add delay between requests to avoid hitting rate limits
	requestDelay := 200 * time.Millisecond

	for {
		// Check if we've reached the limit specified by $top
		if maxResults > 0 && len(allResults) >= maxResults {
			log.Debug().
				Int("max_results", maxResults).
				Int("current_results", len(allResults)).
				Msg("Reached $top limit, stopping pagination")
			// Trim results to exact limit
			allResults = allResults[:maxResults]
			break
		}

		// Add $skip if we're paginating manually
		if skipCount > 0 && len(allResults) > 0 {
			baseEndpoint := currentEndpoint
			if strings.Contains(currentEndpoint, "?") {
				baseEndpoint = strings.Split(currentEndpoint, "?")[0]
			}
			// Preserve all original query parameters (filter, select, orderby, top) when adding $skip
			queryParams := []string{}

			// Parse existing query parameters to preserve them
			if strings.Contains(currentEndpoint, "?") {
				queryPart := strings.Split(currentEndpoint, "?")[1]
				// Split by & but be careful with URL encoding
				params := strings.Split(queryPart, "&")
				for _, param := range params {
					// Skip existing $skip if present (we'll add our own)
					if strings.HasPrefix(param, "$skip=") {
						continue
					}
					// Preserve all other parameters
					if strings.HasPrefix(param, "$filter=") ||
						strings.HasPrefix(param, "$select=") ||
						strings.HasPrefix(param, "$orderby=") ||
						strings.HasPrefix(param, "$top=") {
						queryParams = append(queryParams, param)
					}
				}
			}

			// Add $skip parameter
			queryParams = append(queryParams, fmt.Sprintf("$skip=%d", skipCount))

			// Rebuild endpoint with all parameters
			if len(queryParams) > 0 {
				currentEndpoint = baseEndpoint + "?" + strings.Join(queryParams, "&")
			} else {
				currentEndpoint = fmt.Sprintf("%s?$skip=%d", baseEndpoint, skipCount)
			}
		}

		log.Debug().
			Int("page", pageNum).
			Int("skip", skipCount).
			Str("endpoint", currentEndpoint).
			Msg("Fetching page")

		// Add delay between requests to respect rate limits (except for first request)
		if pageNum > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(requestDelay):
			}
		}

		resp, err := c.Get(ctx, currentEndpoint)
		if err != nil {
			log.Error().Err(err).
				Int("page", pageNum).
				Msg("Failed to fetch page")
			return nil, fmt.Errorf("failed to fetch page: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Error().Err(err).
				Int("page", pageNum).
				Msg("Failed to read response body")
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var odataResp ODataResponse
		if err := json.Unmarshal(body, &odataResp); err != nil {
			log.Error().Err(err).
				Int("page", pageNum).
				Msg("Failed to parse OData response")
			return nil, fmt.Errorf("failed to parse OData response: %w", err)
		}

		pageResults := len(odataResp.Value)
		allResults = append(allResults, odataResp.Value...)

		log.Debug().
			Int("page", pageNum).
			Int("results_in_page", pageResults).
			Int("total_results", len(allResults)).
			Bool("has_next_link", odataResp.NextLink != "").
			Msg("Page fetched")

		// Check if we've reached the limit specified by $top after adding this page
		if maxResults > 0 && len(allResults) >= maxResults {
			log.Debug().
				Int("max_results", maxResults).
				Int("current_results", len(allResults)).
				Msg("Reached $top limit after fetching page, stopping pagination")
			// Trim results to exact limit
			allResults = allResults[:maxResults]
			break
		}

		// Check for next link
		if odataResp.NextLink != "" {
			// If $top is specified, check if we should continue before following next link
			if maxResults > 0 && len(allResults) >= maxResults {
				log.Debug().
					Int("max_results", maxResults).
					Int("current_results", len(allResults)).
					Msg("Reached $top limit, stopping pagination (next link available but limit reached)")
				allResults = allResults[:maxResults]
				break
			}
			// Extract endpoint from next link (remove base URL)
			nextURL, err := url.Parse(odataResp.NextLink)
			if err != nil {
				log.Error().Err(err).
					Str("next_link", odataResp.NextLink).
					Msg("Failed to parse next link")
				return nil, fmt.Errorf("failed to parse next link: %w", err)
			}
			// Remove base path from next link
			nextPath := strings.TrimPrefix(nextURL.Path, strings.TrimSuffix(c.baseURL, "/"))
			currentEndpoint = nextPath + "?" + nextURL.RawQuery
			skipCount = 0 // Reset skip count when using next link
			pageNum++
		} else {
			// No next link - check if we should continue paginating manually
			// Business Central often doesn't include nextLink even when more data is available

			// If we got no results, we're done
			if len(odataResp.Value) == 0 {
				log.Debug().Msg("No more results, pagination complete")
				break
			}

			// If $top is specified and we've reached it, stop
			if maxResults > 0 && len(allResults) >= maxResults {
				log.Debug().
					Int("max_results", maxResults).
					Int("current_results", len(allResults)).
					Msg("Reached $top limit, stopping pagination")
				allResults = allResults[:maxResults]
				break
			}

			// Business Central typically returns 20 results per page when not limited
			// If we got fewer results than a typical page size, we're likely at the end
			// However, with $filter, the page size can vary, so we'll try one more page
			// to be sure, but track if we get the same or fewer results
			typicalPageSize := 20
			if len(odataResp.Value) < typicalPageSize {
				// We got fewer results than typical - might be the last page
				// Try one more page to confirm, but if we already have results from previous iteration
				// with same count, we're done
				if skipCount > 0 {
					// We're already paginating manually, and got a small page
					// This likely means we're at the end
					log.Debug().
						Int("page_size", len(odataResp.Value)).
						Int("typical_page_size", typicalPageSize).
						Msg("Received smaller page than typical, likely at end of results")
					break
				}
			}

			// Continue manual pagination with $skip
			// This works even with $filter - Business Central supports $skip with filters
			log.Debug().
				Int("results_in_page", len(odataResp.Value)).
				Int("skip_count", skipCount).
				Msg("No nextLink found, continuing manual pagination with $skip")
			skipCount += len(odataResp.Value)
			pageNum++
		}

		// Safety check to avoid infinite loops
		if len(odataResp.Value) == 0 {
			log.Debug().Msg("Empty page received, stopping pagination")
			break
		}
	}

	log.Info().
		Int("total_pages", pageNum-1).
		Int("total_results", len(allResults)).
		Msg("Pagination complete")

	return allResults, nil
}

// Query executes an OData query and returns the results
func (c *Client) Query(ctx context.Context, endpoint string, includePagination bool) ([]map[string]interface{}, error) {
	if includePagination {
		return c.GetPaginated(ctx, endpoint)
	}

	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var odataResp ODataResponse
	if err := json.Unmarshal(body, &odataResp); err != nil {
		return nil, fmt.Errorf("failed to parse OData response: %w", err)
	}

	return odataResp.Value, nil
}
