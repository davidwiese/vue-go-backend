# Trackify Fleet (Backend)

![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Live Demo
[trackifyfleet.pro](https://trackifyfleet.pro)

## Project Overview
The **Trackify Fleet Backend** is a Go-based server application that integrates with the OneStep GPS API to provide real-time vehicle tracking data, user preference management, and report generation capabilities. Built with proper package structure and dependency management, it serves as a robust backend for the Trackify Fleet dashboard.

## Technologies Used
- **Backend Framework:**
    - Go
    - Gorilla WebSocket
    - OneStep GPS API Integration
    - AWS SDK

- **Database:**
    - MySQL on AWS RDS
    - SQL Query Optimization
    - Connection Pooling

- **Deployment:**
    - AWS Elastic Beanstalk
    - Environment Configuration
    - SSL/TLS Security

## Features

### OneStep GPS Integration
- Real-time vehicle data fetching
- Custom report generation
- Efficient API response handling
- Auto-retry mechanisms

### WebSocket Management
- Real-time data broadcasting
- Connection pooling
- Client tracking
- Automatic reconnection handling

### User Preferences
- CRUD operations for preferences
- Transaction support
- Batch operations

### Report Generation
- Asynchronous processing
- Status polling
- Multiple time ranges
- PDF generation and streaming

### Structured Architecture
- Handler-based routing
- Package organization
- Dependency injection
- Error handling

## Package Structure
- **api:** HTTP handlers and routing logic
- **database:** Database setup and operations
- **onestepgps:** OneStep GPS API client
- **websocket:** WebSocket hub and client management
- **config:** Application configuration
- **models:** Data structures and interfaces

## Future Improvements
- Enhanced error logging and monitoring
- Rate limiting and request throttling
- Cache implementation
- Advanced query optimization
- Comprehensive test coverage
- API documentation
- Metrics collection

## License
This project is licensed under the [MIT License](LICENSE).

## Related Repositories
- [Frontend Repository](https://github.com/davidwiese/vue-go-dashboard)
