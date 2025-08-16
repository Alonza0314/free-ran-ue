# AuthenticationApi

All URIs are relative to *http://127.0.0.1:40104*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**apiConsoleAuthenticatePost**](#apiconsoleauthenticatepost) | **POST** /api/console/authenticate | Authenticate Token|
|[**apiConsoleLoginPost**](#apiconsoleloginpost) | **POST** /api/console/login | User Login|
|[**apiConsoleLogoutDelete**](#apiconsolelogoutdelete) | **DELETE** /api/console/logout | User Logout|

# **apiConsoleAuthenticatePost**
> ApiConsoleAuthenticatePost200Response apiConsoleAuthenticatePost()

Validate JWT token

### Example

```typescript
import {
    AuthenticationApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new AuthenticationApi(configuration);

const { status, data } = await apiInstance.apiConsoleAuthenticatePost();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**ApiConsoleAuthenticatePost200Response**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | Authentication successful |  -  |
|**401** | Authentication failed |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **apiConsoleLoginPost**
> ApiConsoleLoginPost200Response apiConsoleLoginPost(apiConsoleLoginPostRequest)

Authenticate user and return JWT token

### Example

```typescript
import {
    AuthenticationApi,
    Configuration,
    ApiConsoleLoginPostRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new AuthenticationApi(configuration);

let apiConsoleLoginPostRequest: ApiConsoleLoginPostRequest; //

const { status, data } = await apiInstance.apiConsoleLoginPost(
    apiConsoleLoginPostRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **apiConsoleLoginPostRequest** | **ApiConsoleLoginPostRequest**|  | |


### Return type

**ApiConsoleLoginPost200Response**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | Login successful |  -  |
|**400** | Invalid request format |  -  |
|**401** | Invalid credentials |  -  |
|**500** | Internal server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **apiConsoleLogoutDelete**
> ApiConsoleLogoutDelete200Response apiConsoleLogoutDelete()

Logout current user

### Example

```typescript
import {
    AuthenticationApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new AuthenticationApi(configuration);

const { status, data } = await apiInstance.apiConsoleLogoutDelete();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**ApiConsoleLogoutDelete200Response**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | Logout successful |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

