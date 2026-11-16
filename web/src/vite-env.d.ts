/// <reference types="vite/client" />

declare module "monaco-editor/languages/definitions/go/register.js";
declare module "monaco-editor/editor/editor.worker.js?worker" {
  const WorkerFactory: new () => Worker;
  export default WorkerFactory;
}
