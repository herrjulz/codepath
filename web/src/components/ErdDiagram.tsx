import dagre from "@dagrejs/dagre";
import { PanZoomViewport } from "./PanZoomViewport";
import type { ErdDiagram as ErdData, ErdEntity, ErdRelationship } from "../lib/schema";

const entityWidth = 200;
const attrRow = 18;
const headerHeight = 32;
const padX = 12;
const padY = 10;

export function ErdDiagram({ erd }: { erd: ErdData }) {
  const layout = layoutErd(erd);

  return (
    <PanZoomViewport contentWidth={layout.width} contentHeight={layout.height} label="ERD canvas">
      <svg
        className="erd-svg"
        width={layout.width}
        height={layout.height}
        viewBox={`0 0 ${layout.width} ${layout.height}`}
        role="img"
        aria-label="Entity relationship diagram"
      >
        <defs>
          <marker id="erd-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" className="erd-marker" />
          </marker>
        </defs>
        {layout.edges.map((edge) => (
          <path key={`edge-${edge.id}`} className="erd-edge" d={edge.path} markerEnd="url(#erd-arrow)" />
        ))}
        {layout.nodes.map((node) => (
          <EntityBox key={node.id} entity={node.entity} x={node.x} y={node.y} height={node.height} />
        ))}
        {layout.edges.map((edge) =>
          edge.label ? (
            <g key={`label-${edge.id}`} className="erd-label-group">
              <rect
                className="erd-edge-label-bg"
                x={edge.labelX - edge.labelWidth / 2}
                y={edge.labelY - 11}
                width={edge.labelWidth}
                height={18}
                rx={4}
              />
              <text className="erd-edge-label" x={edge.labelX} y={edge.labelY + 2} textAnchor="middle">
                {edge.label}
              </text>
            </g>
          ) : null,
        )}
      </svg>
    </PanZoomViewport>
  );
}

function EntityBox({ entity, x, y, height }: { entity: ErdEntity; x: number; y: number; height: number }) {
  const attrs = entity.attributes ?? [];
  return (
    <g transform={`translate(${x}, ${y})`}>
      <rect className="erd-box" width={entityWidth} height={height} rx={8} />
      <rect className="erd-box-head" width={entityWidth} height={headerHeight} rx={8} />
      <rect className="erd-box-head-square" y={headerHeight - 8} width={entityWidth} height={8} />
      <text className="erd-entity-label" x={entityWidth / 2} y={21} textAnchor="middle">
        {truncate(entity.label, 22)}
      </text>
      {attrs.map((attr, index) => {
        const ay = headerHeight + padY + index * attrRow + 12;
        const key = attr.key ? `${attr.key.toUpperCase()} ` : "";
        const type = attr.type ? ` : ${attr.type}` : "";
        return (
          <text key={`${attr.name}-${index}`} className="erd-attr" x={padX} y={ay}>
            {truncate(`${key}${attr.name}${type}`, 28)}
          </text>
        );
      })}
    </g>
  );
}

function layoutErd(erd: ErdData) {
  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({ rankdir: "LR", nodesep: 56, ranksep: 110, marginx: 28, marginy: 28 });

  const heights = new Map<string, number>();
  for (const entity of erd.entities) {
    const height = headerHeight + padY * 2 + Math.max(1, entity.attributes?.length ?? 0) * attrRow;
    heights.set(entity.id, height);
    graph.setNode(entity.id, { width: entityWidth, height });
  }
  for (const rel of erd.relationships) {
    graph.setEdge(rel.from, rel.to);
  }
  dagre.layout(graph);

  const nodes = erd.entities.map((entity) => {
    const pos = graph.node(entity.id);
    const height = heights.get(entity.id) ?? headerHeight;
    return {
      id: entity.id,
      entity,
      x: (pos?.x ?? entityWidth / 2) - entityWidth / 2,
      y: (pos?.y ?? height / 2) - height / 2,
      height,
    };
  });

  const edges = erd.relationships.map((rel) => edgeGeometry(rel, nodes));
  const width = Math.max(360, ...(nodes.map((n) => n.x + entityWidth + 24)));
  const height = Math.max(240, ...(nodes.map((n) => n.y + n.height + 24)));
  return { nodes, edges, width, height };
}

function edgeGeometry(
  rel: ErdRelationship,
  nodes: { id: string; x: number; y: number; height: number }[],
) {
  const from = nodes.find((n) => n.id === rel.from);
  const to = nodes.find((n) => n.id === rel.to);
  if (!from || !to) {
    return { id: rel.id, path: "", label: "", labelX: 0, labelY: 0, labelWidth: 0 };
  }
  const x1 = from.x + entityWidth;
  const y1 = from.y + from.height / 2;
  const x2 = to.x;
  const y2 = to.y + to.height / 2;
  const midX = (x1 + x2) / 2;
  const path = `M ${x1} ${y1} C ${midX} ${y1}, ${midX} ${y2}, ${x2} ${y2}`;
  const kind = kindLabel(rel.kind);
  const label = [kind, rel.label].filter(Boolean).join(" · ");
  // Cubic midpoint at t=0.5, then lift above the curve so boxes don't cover it.
  const labelX = 0.125 * x1 + 0.375 * midX + 0.375 * midX + 0.125 * x2;
  const labelY = 0.125 * y1 + 0.375 * y1 + 0.375 * y2 + 0.125 * y2 - 14;
  const labelWidth = Math.max(36, Math.min(160, label.length * 6.2 + 14));
  return {
    id: rel.id,
    path,
    label,
    labelX,
    labelY,
    labelWidth,
  };
}

function kindLabel(kind: ErdRelationship["kind"]) {
  switch (kind) {
    case "one_to_one":
      return "1:1";
    case "one_to_many":
      return "1:N";
    case "many_to_one":
      return "N:1";
    case "many_to_many":
      return "N:N";
    default:
      return kind;
  }
}

function truncate(value: string, max: number) {
  if (value.length <= max) {
    return value;
  }
  return `${value.slice(0, max - 1)}…`;
}
