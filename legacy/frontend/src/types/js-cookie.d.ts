declare module "js-cookie" {
  interface CookiesStatic {
    get(name?: string): string | undefined;
    set(name: string, value: string | object, options?: object): string | undefined;
    remove(name: string, options?: object): void;
  }
  const Cookies: CookiesStatic;
  export default Cookies;
}
