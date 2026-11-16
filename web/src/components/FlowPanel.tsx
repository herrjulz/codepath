import {
  Background,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import { useEffect, useMemo } from "react";
import "@xyflow/react/dist/style.css";
import { adjacentStep, stepPosition } from "../lib/navigation";
import type { FlowEdge, FlowStep, Scenario } from "../lib/schema";
import { ScenarioSelector } from "./ScenarioSelector";

const nodeWidth = 200;
const nodeHeight = 44;
const edgeColor = "#93a0b0";
const edgeHotColor = "#8eb4ff";

type StepData = {
  title: string;
  active: boolean;
  onPath: boolean;
};

type StepNode = Node<StepData, "step">;

const nodeTypes = { step: StepNodeView };

export function FlowPanel({
  steps,
  edges,
  scenarios,
  scenario,
  stepId,
  onScenario,
  onSelect,
}: {
  steps: FlowStep[];
  edges: FlowEdge[];
  scenarios: Scenario[];
  scenario: Scenario;
  stepId: string;
  onScenario: (id: string) => void;
  onSelect: (stepId: string) => void;
}) {
  const index = stepPosition(scenario.path, stepId);
  const previous = adjacentStep(scenario.path, stepId, -1);
  const next = adjacentStep(scenario.path, stepId, 1);
  const position = index < 0 ? "Off path" : `Step ${index + 1} of ${scenario.path.length}`;

  return (
    <section className="col flow" aria-label="Flow">
      <header className="col-head flow-head">
        <div className="flow-tools">
          <ScenarioSelector scenarios={scenarios} value={scenario.id} onChange={onScenario} />
          <p className="step-count">{position}</p>
        </div>
        <div className="nav-buttons">
          <button type="button" onClick={() => previous && onSelect(previous)} disabled={!previous}>
            Previous
          </button>
          <button type="button" onClick={() => next && onSelect(next)} disabled={!next}>
            Next
          </button>
          {index < 0 ? (
            <button type="button" onClick={() => onSelect(scenario.path[0])}>
              Return to path
            </button>
          ) : null}
        </div>
      </header>
      {scenario.description ? <p className="scenario-copy">{scenario.description}</p> : null}
      <div className="flow-canvas">
        <ReactFlowProvider>
          <FlowCanvas
            steps={steps}
            edges={edges}
            scenario={scenario}
            stepId={stepId}
            onSelect={onSelect}
          />
          <FitButton scenarioId={scenario.id} />
        </ReactFlowProvider>
      </div>
    </section>
  );
}

function FlowCanvas({
  steps,
  edges,
  scenario,
  stepId,
  onSelect,
}: {
  steps: FlowStep[];
  edges: FlowEdge[];
  scenario: Scenario;
  stepId: string;
  onSelect: (stepId: string) => void;
}) {
  const graph = useMemo(() => layout(steps, edges, scenario.path, stepId), [steps, edges, scenario.path, stepId]);

  return (
    <ReactFlow
      nodes={graph.nodes}
      edges={graph.edges}
      nodeTypes={nodeTypes}
      onNodeClick={(_, node) => onSelect(node.id)}
      fitView
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable={false}
      panOnScroll
      minZoom={0.15}
      maxZoom={1.6}
      colorMode="system"
      proOptions={{ hideAttribution: true }}
    >
      <Background gap={18} color="var(--line)" />
      <Controls showInteractive={false} />
      <MiniMap
        pannable
        zoomable
        bgColor="#212938"
        maskColor="rgba(10, 14, 20, 0.62)"
        nodeStrokeColor="#313b4b"
        nodeStrokeWidth={1}
        nodeBorderRadius={4}
        nodeColor={(node) => {
          const data = node.data as StepData | undefined;
          if (data?.active) {
            return "#8eb4ff";
          }
          if (data?.onPath) {
            return "#4a628a";
          }
          return "#3a465a";
        }}
      />
    </ReactFlow>
  );
}

function FitButton({ scenarioId }: { scenarioId: string }) {
  const { fitView } = useReactFlow();
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void fitView({ padding: 0.2, duration: 180 });
    }, 60);
    return () => window.clearTimeout(timer);
  }, [scenarioId, fitView]);
  return (
    <button type="button" className="fit-button" onClick={() => void fitView({ padding: 0.18 })}>
      Fit
    </button>
  );
}

function StepNodeView({ data }: NodeProps<StepNode>) {
  const classes = ["step-node"];
  if (data.onPath) {
    classes.push("on-path");
  }
  if (data.active) {
    classes.push("active");
  }
  return (
    <div className={classes.join(" ")}>
      <Handle type="target" position={Position.Top} className="step-handle" isConnectable={false} />
      <div className="step-title">{data.title}</div>
      <Handle type="source" position={Position.Bottom} className="step-handle" isConnectable={false} />
    </div>
  );
}

function layout(steps: FlowStep[], edges: FlowEdge[], path: string[], activeId: string): {
  nodes: StepNode[];
  edges: Edge[];
} {
  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({ rankdir: "TB", nodesep: 36, ranksep: 64, marginx: 12, marginy: 12 });
  for (const step of steps) {
    graph.setNode(step.id, { width: nodeWidth, height: nodeHeight });
  }
  for (const edge of edges) {
    graph.setEdge(edge.from, edge.to);
  }
  dagre.layout(graph);
  const onPath = new Set(path);
  const pairs = new Set<string>();
  for (let i = 0; i < path.length - 1; i += 1) {
    pairs.add(`${path[i]}>${path[i + 1]}`);
  }
  return {
    nodes: steps.map((step) => {
      const position = graph.node(step.id) ?? { x: nodeWidth / 2, y: nodeHeight / 2 };
      return {
        id: step.id,
        type: "step",
        position: { x: position.x - nodeWidth / 2, y: position.y - nodeHeight / 2 },
        width: nodeWidth,
        height: nodeHeight,
        data: {
          title: step.title,
          active: step.id === activeId,
          onPath: onPath.has(step.id),
        },
        draggable: false,
        connectable: false,
      };
    }),
    edges: edges.map((edge) => {
      const hot = pairs.has(`${edge.from}>${edge.to}`);
      const color = hot ? edgeHotColor : edgeColor;
      return {
        id: edge.id,
        source: edge.from,
        target: edge.to,
        label: edge.label,
        type: "smoothstep",
        animated: hot,
        style: { stroke: color, strokeWidth: hot ? 2.5 : 1.5 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 18,
          height: 18,
          color,
        },
        labelStyle: { fill: "var(--text)", fontSize: 11 },
        labelBgStyle: { fill: "var(--panel)", fillOpacity: 0.9 },
        labelBgPadding: [4, 2] as [number, number],
        labelBgBorderRadius: 4,
      };
    }),
  };
}
