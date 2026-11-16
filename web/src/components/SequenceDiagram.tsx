import type { KeyboardEvent } from "react";
import { PanZoomViewport } from "./PanZoomViewport";
import type { SequenceDiagram as SequenceData, SequenceMessage } from "../lib/schema";

const colWidth = 160;
const leftPad = 32;
const topPad = 64;
const rowHeight = 56;
const bottomPad = 32;
const headHeight = 34;
const headHalf = 58;

export function sequenceSize(sequence: SequenceData) {
  const width = Math.max(leftPad * 2 + sequence.participants.length * colWidth, 360);
  const height = topPad + sequence.messages.length * rowHeight + bottomPad;
  return { width, height };
}

export function SequenceDiagram({
  sequence,
  onSelectStep,
}: {
  sequence: SequenceData;
  onSelectStep: (stepId: string) => void;
}) {
  const { participants, messages } = sequence;
  const { width, height } = sequenceSize(sequence);
  const xFor = (id: string) => {
    const index = participants.findIndex((item) => item.id === id);
    return leftPad + colWidth / 2 + Math.max(0, index) * colWidth;
  };

  return (
    <PanZoomViewport contentWidth={width} contentHeight={height} label="Sequence canvas">
      <svg
        className="sequence-svg"
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        aria-label="Sequence diagram"
      >
        <defs>
          <marker id="seq-arrow-call" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" className="sequence-marker call" />
          </marker>
          <marker id="seq-arrow-return" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" className="sequence-marker return" />
          </marker>
        </defs>
        {participants.map((participant, index) => {
          const x = leftPad + colWidth / 2 + index * colWidth;
          return (
            <g key={participant.id}>
              <rect className="sequence-head" x={x - headHalf} y={8} width={headHalf * 2} height={headHeight} rx={8} />
              <text className="sequence-head-label" x={x} y={30} textAnchor="middle">
                {truncate(participant.label, 16)}
              </text>
              <line className="sequence-lifeline" x1={x} y1={8 + headHeight} x2={x} y2={height - 12} />
            </g>
          );
        })}
        {messages.map((message, index) => (
          <MessageRow
            key={message.id}
            message={message}
            y={topPad + index * rowHeight}
            xFrom={xFor(message.from)}
            xTo={xFor(message.to)}
            onSelectStep={onSelectStep}
          />
        ))}
      </svg>
    </PanZoomViewport>
  );
}

function MessageRow({
  message,
  y,
  xFrom,
  xTo,
  onSelectStep,
}: {
  message: SequenceMessage;
  y: number;
  xFrom: number;
  xTo: number;
  onSelectStep: (stepId: string) => void;
}) {
  const clickable = Boolean(message.stepId);
  const onActivate = () => {
    if (message.stepId) {
      onSelectStep(message.stepId);
    }
  };
  const onKey = (event: KeyboardEvent<SVGGElement>) => {
    if (!clickable) {
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onActivate();
    }
  };

  if (message.kind === "note") {
    return (
      <g
        className={clickable ? "sequence-msg clickable" : "sequence-msg"}
        role={clickable ? "button" : undefined}
        tabIndex={clickable ? 0 : undefined}
        onClick={onActivate}
        onKeyDown={onKey}
      >
        <rect className="sequence-note" x={xFrom - 60} y={y - 14} width={120} height={32} rx={6} />
        <text className="sequence-label" x={xFrom} y={y + 6} textAnchor="middle">
          {truncate(message.label, 22)}
        </text>
      </g>
    );
  }

  const left = Math.min(xFrom, xTo);
  const right = Math.max(xFrom, xTo);
  const forward = xTo >= xFrom;
  const marker = message.kind === "return" ? "url(#seq-arrow-return)" : "url(#seq-arrow-call)";
  const strokeClass = message.kind === "return" ? "sequence-arrow return" : "sequence-arrow call";

  return (
    <g
      className={clickable ? "sequence-msg clickable" : "sequence-msg"}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      onClick={onActivate}
      onKeyDown={onKey}
    >
      <line
        className={strokeClass}
        x1={forward ? left : right}
        y1={y}
        x2={forward ? right : left}
        y2={y}
        markerEnd={marker}
      />
      <text className="sequence-label" x={(left + right) / 2} y={y - 8} textAnchor="middle">
        {truncate(message.label, 28)}
      </text>
    </g>
  );
}

function truncate(value: string, max: number) {
  if (value.length <= max) {
    return value;
  }
  return `${value.slice(0, max - 1)}…`;
}
