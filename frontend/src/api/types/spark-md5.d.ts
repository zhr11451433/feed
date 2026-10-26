declare module 'spark-md5' {
    class SparkMD5 {
        static ArrayBuffer: {
            new (): {
                append(arr: ArrayBuffer): void
                end(): string
            }
        }
    }
    export default SparkMD5
}