import { AuthenticationApi, GNBApi } from './api'
import axios from 'axios'

const getConsoleUrl = () => {
    const { protocol, hostname } = window.location
    return `${protocol}//${hostname}:40104`
}

const apiConfig = {
    basePath: getConsoleUrl(),
    isJsonMime: () => false,
}

export const consoleApi = new AuthenticationApi(apiConfig, undefined, axios)
export const gnbApi = new GNBApi(apiConfig, undefined, axios)
