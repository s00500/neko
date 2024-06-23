/* tslint:disable */
/* eslint-disable */
/**
 * n.eko REST API
 * Next Gen Renderer.
 *
 * The version of the OpenAPI document: 1.0.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */



/**
 * 
 * @export
 * @interface BatchResponse
 */
export interface BatchResponse {
    /**
     * 
     * @type {string}
     * @memberof BatchResponse
     */
    'path'?: string;
    /**
     * 
     * @type {string}
     * @memberof BatchResponse
     */
    'method'?: BatchResponseMethodEnum;
    /**
     * Response body
     * @type {any}
     * @memberof BatchResponse
     */
    'body'?: any;
    /**
     * 
     * @type {number}
     * @memberof BatchResponse
     */
    'status'?: number;
}

export const BatchResponseMethodEnum = {
    GET: 'GET',
    POST: 'POST',
    DELETE: 'DELETE'
} as const;

export type BatchResponseMethodEnum = typeof BatchResponseMethodEnum[keyof typeof BatchResponseMethodEnum];


