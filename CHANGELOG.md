# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial MCP server implementation for Business Central OData APIs
- OAuth 2.0 authentication with token caching and refresh
- OData query support with filtering, sorting, and pagination
- Entity retrieval by key
- Count endpoint support
- Automatic pagination handling
- Retry logic with exponential backoff
- Rate limiting support

### Fixed
- Respect `$top` parameter in pagination queries
- Handle pagination when `nextLink` is missing from OData responses

[Unreleased]: https://github.com/iafnetworkspa/bc-odata-mcp/compare/v0.1.0...HEAD

