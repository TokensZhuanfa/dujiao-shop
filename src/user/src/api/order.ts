import { userApi } from './client'
import type { CreatePaymentPayload } from './types'

export const userOrderAPI = {
    preview: (data: any) => userApi.post('/orders/preview', data),
    getPaymentChannels: (data: any) => userApi.post('/order/payment-channels', data),
    create: (data: any) => userApi.post('/orders', data),
    createAndPay: (data: any) => userApi.post('/orders/create-and-pay', data),
    list: (params?: any) => userApi.get('/orders', { params }),
    detail: (orderNo: string, options?: any) => userApi.get(`/orders/${encodeURIComponent(orderNo)}`, options),
    cancel: (orderNo: string) => userApi.post(`/orders/${encodeURIComponent(orderNo)}/cancel`),
    downloadFulfillment: (orderNo: string) => userApi.get(`/orders/${encodeURIComponent(orderNo)}/fulfillment/download`, { blob: true }),
    listFulfillmentFiles: (orderNo: string) => userApi.get(`/orders/${encodeURIComponent(orderNo)}/fulfillment/files`),
    downloadFulfillmentFile: (orderNo: string, fileId: string) => userApi.get(`/orders/${encodeURIComponent(orderNo)}/fulfillment/files/${encodeURIComponent(fileId)}/download`, { blob: true }),
    downloadFulfillmentArchive: (orderNo: string) => userApi.get(`/orders/${encodeURIComponent(orderNo)}/fulfillment/files/archive`, { blob: true }),
    listCodexAccounts: (orderNo: string) => userApi.get(`/orders/${encodeURIComponent(orderNo)}/codex-accounts`),
    downloadCodexAccount: (orderNo: string, format: 'cpamc' | 'sub2api', id: number) =>
        userApi.get(`/orders/${encodeURIComponent(orderNo)}/codex-accounts/${format}/${id}`, { blob: true }),
    downloadCodexAccountsArchive: (orderNo: string, format: 'cpamc' | 'sub2api') =>
        userApi.get(`/orders/${encodeURIComponent(orderNo)}/codex-accounts/${format}/archive`, { blob: true }),
}

export const guestOrderAPI = {
    preview: (data: any) => userApi.post('/guest/orders/preview', data),
    create: (data: any) => userApi.post('/guest/orders', data),
    createAndPay: (data: any) => userApi.post('/guest/orders/create-and-pay', data),
    list: (params: any) => userApi.get('/guest/orders', { params }),
    detail: (orderNo: string, params: any, options?: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}`, { params, ...(options || {}) }),
    downloadFulfillment: (orderNo: string, params: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/fulfillment/download`, { params, blob: true }),
    listFulfillmentFiles: (orderNo: string, params: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/fulfillment/files`, { params }),
    downloadFulfillmentFile: (orderNo: string, fileId: string, params: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/fulfillment/files/${encodeURIComponent(fileId)}/download`, { params, blob: true }),
    downloadFulfillmentArchive: (orderNo: string, params: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/fulfillment/files/archive`, { params, blob: true }),
    listCodexAccounts: (orderNo: string, params: any) => userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/codex-accounts`, { params }),
    downloadCodexAccount: (orderNo: string, format: 'cpamc' | 'sub2api', id: number, params: any) =>
        userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/codex-accounts/${format}/${id}`, { params, blob: true }),
    downloadCodexAccountsArchive: (orderNo: string, format: 'cpamc' | 'sub2api', params: any) =>
        userApi.get(`/guest/orders/${encodeURIComponent(orderNo)}/codex-accounts/${format}/archive`, { params, blob: true }),
    createPayment: (data: any) => userApi.post('/guest/payments', data),
    capturePayment: (id: number, data: any) => userApi.post(`/guest/payments/${id}/capture`, data),
    latestPayment: (params: any) => userApi.get('/guest/payments/latest', { params, silentBusinessError: true }),
}

export const paymentAPI = {
    create: (data: CreatePaymentPayload) => userApi.post('/payments', data),
    capture: (id: number) => userApi.post(`/payments/${id}/capture`),
    latest: (params: any) => userApi.get('/payments/latest', { params, silentBusinessError: true }),
}
