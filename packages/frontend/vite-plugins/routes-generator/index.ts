import fs from 'node:fs'
import path from 'node:path'

import type { Plugin } from 'vite'

/** A source directory that contributes pages to the generated SPA route table. */
export interface RouteRoot {
  moduleId: string
  directory: string
  basePath?: string
}

/** Configuration for the workspace-wide route generator. */
export interface Options {
  layoutDir?: string
  routeRoots?: RouteRoot[]
}

interface RouteMeta {
  title: string
  permission?: string
  [key: string]: unknown
}

interface RouterConfig {
  isRouter?: boolean
  param?: string
  routePath?: string
  [key: string]: unknown
}

type RouteMetaConfig = Record<string, unknown>

interface RouteRecord {
  path: string
  name: string
  meta: RouteMeta
  props: boolean
  alias?: string | string[]
  redirect?: string
  component?: string
  children?: RouteRecord[]
}

const virtualModuleId = 'virtual:router'
const resolvedVirtualModuleId = `\0${virtualModuleId}`

export function routesGeneratorPlugin(options: Options = {}): Plugin {
  const layoutDir = options.layoutDir || '/src/views/Layout.vue'

  return {
    name: 'routes-generator-plugin',
    resolveId(id) {
      if (id === virtualModuleId) {
        return resolvedVirtualModuleId
      }
    },
    load(id) {
      if (id === resolvedVirtualModuleId) {
        return `import Layout from ${JSON.stringify(layoutDir)}\nexport default ${getCode({ ...options, layoutDir })}`
      }
    },
    handleHotUpdate({ file, server }) {
      const routeRoots = getRouteRoots(options)
      if (routeRoots.some((root) => filePathIsInDirectory(file, root.directory))) {
        const module = server.moduleGraph.getModuleById(resolvedVirtualModuleId)
        if (module) server.moduleGraph.invalidateModule(module)
        server.ws.send({ type: 'full-reload' })
      }
    }
  }
}

/** Generate executable JavaScript for the virtual route module. */
function getCode(options: Options): string {
  return serializeRouteValue(getGeneratedRouteRecords(options), '  ', 0)
}

/** Generate route records from all configured application and package roots. */
export function getGeneratedRouteRecords(options: Options = {}): RouteRecord[] {
  const routeRoots = getRouteRoots(options)
  const routes: RouteRecord[] = []
  const routeMap: Record<string, RouteRecord> = {}
  const routeOwners: Record<string, string> = {}

  validateRouteRoots(routeRoots)

  for (const root of routeRoots) {
    const files = collectViewFiles(root.directory)
      .map((file) => file.replace(root.directory, '').split(path.sep).join('/'))
      .sort()
    const directoryMetaByPath = collectDirectoryMetaByPath(root.directory)
    generateRoutes(
      files,
      directoryMetaByPath,
      root,
      routes,
      routeMap,
      routeOwners,
    )
  }

  return routes
}

/** Validate root identity and prevent two packages from silently owning one path. */
function validateRouteRoots(routeRoots: RouteRoot[]) {
  const moduleIds = new Set<string>()
  for (const root of routeRoots) {
    if (moduleIds.has(root.moduleId)) throw new Error(`Duplicate route moduleId: ${root.moduleId}`)
    moduleIds.add(root.moduleId)
    if (!fs.existsSync(root.directory)) throw new Error(`Route root does not exist: ${root.directory}`)
  }
}

function getRouteRoots(options: Options): RouteRoot[] {
  if (options.routeRoots?.length) return options.routeRoots

  return [{
    moduleId: 'app',
    directory: path.resolve(process.cwd(), './src/views'),
    basePath: '',
  }]
}

function collectViewFiles(root: string): string[] {
  const result: string[] = []
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const fullPath = path.join(root, entry.name)
    if (entry.isDirectory()) {
      if (isComponentsDirectory(entry.name)) continue
      result.push(...collectViewFiles(fullPath))
      continue
    }
    if (!/\.(vue|tsx|jsx)$/.test(entry.name)) continue
    if (['Layout.vue', 'Signin.vue', '403.vue', '404.vue'].includes(entry.name)) continue
    result.push(fullPath)
  }
  return result
}

function collectDirectoryMetaByPath(root: string, current = root, result: Record<string, RouteMetaConfig> = {}) {
  for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
    const fullPath = path.join(current, entry.name)
    if (entry.isDirectory()) {
      if (isComponentsDirectory(entry.name)) continue
      collectDirectoryMetaByPath(root, fullPath, result)
      continue
    }
    if (entry.name !== 'routeMeta.json') continue
    result[toRoutePathFromDirectory(root, current)] = readRouteMetaJson(fullPath)
  }
  return result
}

function isComponentsDirectory(name: string) {
  return name.toLowerCase() === 'components'
}

function readRouteMetaJson(file: string): RouteMetaConfig {
  const content = fs.readFileSync(file, 'utf-8')
  const meta = JSON.parse(content) as unknown
  if (!meta || typeof meta !== 'object' || Array.isArray(meta)) {
    throw new Error(`${file} must contain a JSON object`)
  }
  return meta as RouteMetaConfig
}

function toRoutePathFromDirectory(root: string, directory: string) {
  const relativePath = path.relative(root, directory).split(path.sep).join('/').toLowerCase()
  return toFullPath(relativePath.split('/').filter(Boolean))
}

function generateRoutes(
  files: string[],
  directoryMetaByPath: Record<string, RouteMetaConfig>,
  root: RouteRoot,
  routes: RouteRecord[],
  map: Record<string, RouteRecord>,
  routeOwners: Record<string, string>,
) {
  for (const file of files) {
    const fullPath = path.resolve(root.directory, `.${file}`)
    const content = fs.readFileSync(fullPath, 'utf-8')
    const routerConfig = readRouterConfig(content)
    if (routerConfig.isRouter === false) continue

    const { param, routePath, ...customMeta } = routerConfig
    const localParts = routePath ? toRoutePartsFromPath(routePath) : toRouteParts(file)
    const parts = [...toRoutePartsFromPath(root.basePath || ''), ...localParts]
    if (!parts.length) {
      throw new Error(`Route file ${fullPath} must resolve to a non-empty path`)
    }
    const aliases = routePath ? [] : kebabAliases(file, parts, param)
    const component = `() => import(${JSON.stringify(fullPath)})`
    ensureRoute(
      routes,
      map,
      directoryMetaByPath,
      root,
      parts,
      component,
      param,
      customMeta,
      aliases,
      routeOwners,
    )
  }
}

function ensureRoute(
  routes: RouteRecord[],
  map: Record<string, RouteRecord>,
  directoryMetaByPath: Record<string, RouteMetaConfig>,
  root: RouteRoot,
  parts: string[],
  component: string,
  param: string | undefined,
  customMeta: Record<string, unknown>,
  aliases: string[],
  routeOwners: Record<string, string>,
) {
  const leafPath = toFullPath(parts)
  const isTopLevel = parts.length === 1

  if (isTopLevel) {
    const route = getOrCreateRoute(routes, map, directoryMetaByPath, root, parts, routeOwners)
    route.component = 'Layout'
    if (aliases.length) route.alias = aliases
    delete route.redirect
    setIndexChild(route, {
      path: '',
      name: `${root.moduleId}-${parts.join('-')}-index`,
      meta: buildRouteMeta(directoryMetaByPath, leafPath, parts, customMeta),
      props: true,
      component,
    })
    return
  }

  for (let index = 0; index < parts.length; index += 1) {
    const currentParts = parts.slice(0, index + 1)
    const currentPath = toFullPath(currentParts)
    const isLeaf = currentPath === leafPath
    const route = getOrCreateRoute(routes, map, directoryMetaByPath, root, currentParts, routeOwners)

    route.meta = {
      ...buildRouteMeta(directoryMetaByPath, currentPath, currentParts),
      ...(isLeaf ? customMeta : {})
    }

    if (index === 0) {
      route.component = 'Layout'
      if (!hasIndexChild(route)) {
        route.redirect = leafPath
      }
    } else if (isLeaf) {
      route.path = currentPath + (param ? `/${param}` : '')
      if (aliases.length) route.alias = aliases
      route.component = component
    }
  }
}

function getOrCreateRoute(
  routes: RouteRecord[],
  map: Record<string, RouteRecord>,
  directoryMetaByPath: Record<string, RouteMetaConfig>,
  root: RouteRoot,
  parts: string[],
  routeOwners: Record<string, string>,
) {
  const currentPath = toFullPath(parts)
  const existing = map[currentPath]
  if (existing) {
    if (routeOwners[currentPath] !== root.moduleId) {
      throw new Error(`Route path ${currentPath} is declared by both ${routeOwners[currentPath]} and ${root.moduleId}`)
    }
    return existing
  }

  const route: RouteRecord = {
    path: currentPath,
    name: `${root.moduleId}-${parts.join('-')}`,
    meta: buildRouteMeta(directoryMetaByPath, currentPath, parts),
    props: true,
    children: []
  }

  if (parts.length === 1) {
    routes.push(route)
  } else {
    const parent = map[toFullPath(parts.slice(0, -1))]
    parent?.children?.push(route)
  }

  map[currentPath] = route
  routeOwners[currentPath] = root.moduleId
  return route
}

function hasIndexChild(route: RouteRecord) {
  return route.children?.some((child) => child.path === '') ?? false
}

function setIndexChild(route: RouteRecord, child: RouteRecord) {
  const children = route.children || []
  const index = children.findIndex((item) => item.path === '')
  if (index >= 0) {
    children[index] = child
  } else {
    children.unshift(child)
  }
  route.children = children
}

function toRouteParts(file: string) {
  const routePath = file.replace(/\.(vue|tsx|jsx)$/, '').toLowerCase()
  const parts = routePath.split('/').filter(Boolean)
  return parts[parts.length - 1] === 'index' ? parts.slice(0, -1) : parts
}

function toRoutePartsFromPath(routePath: string) {
  return routePath
    .trim()
    .replace(/^\/+|\/+$/g, '')
    .split('/')
    .filter(Boolean)
}

function kebabAliases(file: string, canonicalParts: string[], param: string | undefined) {
  const routePath = file.replace(/\.(vue|tsx|jsx)$/, '')
  const parts = routePath.split('/').filter(Boolean).map(toKebabCase)
  const kebabParts = parts[parts.length - 1] === 'index' ? parts.slice(0, -1) : parts
  const canonicalPath = toFullPath(canonicalParts) + (param ? `/${param}` : '')
  const prefix = kebabParts.length ? canonicalParts.slice(0, -kebabParts.length) : canonicalParts
  const kebabPath = toFullPath([...prefix, ...kebabParts]) + (param ? `/${param}` : '')
  return kebabPath === canonicalPath ? [] : [kebabPath]
}

function toKebabCase(value: string) {
  return value
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1-$2')
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[_\s]+/g, '-')
    .toLowerCase()
}

function toFullPath(parts: string[]) {
  return `/${parts.join('/')}`
}

function buildRouteMeta(directoryMetaByPath: Record<string, RouteMetaConfig>, routePath: string, parts: string[], customMeta: Record<string, unknown> = {}) {
  return {
    title: parts.join('.'),
    permission: getRoutePermission(parts),
    ...(directoryMetaByPath[routePath] || {}),
    ...customMeta
  }
}

function readRouterConfig(content: string): RouterConfig {
  const match = content.match(/<router\s+lang="json">([\s\S]*?)<\/router>/)
  if (!match) return {}
  return JSON.parse(match[1] || '{}')
}

function getRoutePermission(parts: string[]) {
  return parts.join(':') + ':view'
}

function serializeRouteValue(value: unknown, indent: string, level: number): string {
  if (value === 'Layout') return 'Layout'
  if (typeof value === 'string' && value.startsWith('() => import(')) return value
  if (Array.isArray(value)) {
    if (!value.length) return '[]'
    const padding = indent.repeat(level + 1)
    const closingPadding = indent.repeat(level)
    return `[\n${padding}${value.map((item) => serializeRouteValue(item, indent, level + 1)).join(`,\n${padding}`)}\n${closingPadding}]`
  }
  if (value && typeof value === 'object') {
    const entries = Object.entries(value)
    if (!entries.length) return '{}'
    const padding = indent.repeat(level + 1)
    const closingPadding = indent.repeat(level)
    return `{\n${padding}${entries.map(([key, item]) => `${JSON.stringify(key)}: ${serializeRouteValue(item, indent, level + 1)}`).join(`,\n${padding}`)}\n${closingPadding}}`
  }
  return JSON.stringify(value)
}

function filePathIsInDirectory(file: string, directory: string) {
  const relative = path.relative(directory, file)
  return relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))
}
