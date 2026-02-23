import { extent, max, min } from "d3-array";
import { axisBottom, axisLeft } from "d3-axis";
import { format } from "d3-format";
import { scaleLinear, scaleTime } from "d3-scale";
import { pointer, select } from "d3-selection";
import { line as d3Line } from "d3-shape";
import { timeWeek } from "d3-time";
import { timeFormat, timeParse } from "d3-time-format";

import { rawRows } from "./weights.generated.js";

const parseDate = timeParse("%Y-%m-%d");
const formatDate = timeFormat("%b %-d");
const formatTooltipDate = timeFormat("%A, %b %-d, %Y");
const formatWeight = format(".1f");

const rows = rawRows.map((d) => ({
  date: parseDate(d.date),
  weight: d.weight_lbs == null ? null : Number(d.weight_lbs),
  injectionDate: d.injection_date ? parseDate(d.injection_date) : null,
  dose: d.dose || null,
}));

const weights = rows.filter((d) => d.weight != null);
const injections = rows.filter((d) => d.injectionDate != null);

const svg = select("#chart");
const tooltip = select("#chart-tooltip");
const cardNode = document.querySelector(".card");

if (svg.empty() || tooltip.empty() || !cardNode) {
  throw new Error("zepbound chart mount points not found");
}

const width = 1000;
const height = 520;
const margin = { top: 64, right: 32, bottom: 56, left: 70 };

const hideTooltip = () => {
  tooltip.style("opacity", 0).attr("aria-hidden", "true");
};

const positionTooltip = (event) => {
  const [mouseX, mouseY] = pointer(event, cardNode);
  const tooltipNode = tooltip.node();
  const offset = 12;
  const padding = 8;

  let left = mouseX + offset;
  let top = mouseY + offset;

  const maxLeft = cardNode.clientWidth - tooltipNode.offsetWidth - padding;
  const maxTop = cardNode.clientHeight - tooltipNode.offsetHeight - padding;

  left = Math.min(Math.max(padding, left), Math.max(padding, maxLeft));
  top = Math.min(Math.max(padding, top), Math.max(padding, maxTop));

  tooltip.style("left", `${left}px`).style("top", `${top}px`);
};

const showTooltip = (event, d) => {
  tooltip
    .html(`${formatTooltipDate(d.date)}<br>${formatWeight(d.weight)} lbs`)
    .style("opacity", 1)
    .attr("aria-hidden", "false");
  positionTooltip(event);
};

const x = scaleTime()
  .domain(extent(rows, (d) => d.date))
  .range([margin.left, width - margin.right]);

const minWeight = min(weights, (d) => d.weight);
const maxWeight = max(weights, (d) => d.weight);

const y = scaleLinear()
  .domain([minWeight - 1.2, maxWeight + 1.2])
  .nice()
  .range([height - margin.bottom, margin.top]);

svg
  .append("g")
  .attr("class", "grid")
  .attr("transform", `translate(${margin.left},0)`)
  .call(axisLeft(y).ticks(6).tickSize(-(width - margin.left - margin.right)).tickFormat(""));

svg
  .append("g")
  .attr("class", "axis")
  .attr("transform", `translate(0,${height - margin.bottom})`)
  .call(axisBottom(x).ticks(timeWeek.every(1)).tickFormat(formatDate));

svg
  .append("g")
  .attr("class", "axis")
  .attr("transform", `translate(${margin.left},0)`)
  .call(axisLeft(y));

svg
  .append("g")
  .selectAll("line")
  .data(injections)
  .join("line")
  .attr("class", "inj-line")
  .attr("x1", (d) => x(d.injectionDate))
  .attr("x2", (d) => x(d.injectionDate))
  .attr("y1", margin.top)
  .attr("y2", height - margin.bottom);

svg
  .append("g")
  .selectAll("text")
  .data(injections)
  .join("text")
  .attr("class", "inj-label")
  .attr("x", (d) => x(d.injectionDate))
  .attr("y", margin.top - 14)
  .text((d) => d.dose);

const line = d3Line()
  .x((d) => x(d.date))
  .y((d) => y(d.weight));

svg
  .append("path")
  .datum(weights)
  .attr("class", "weight-line")
  .attr("d", line);

svg
  .append("g")
  .selectAll("circle")
  .data(weights)
  .join("circle")
  .attr("class", "weight-dot")
  .attr("cx", (d) => x(d.date))
  .attr("cy", (d) => y(d.weight))
  .attr("r", 2.7);

svg
  .append("g")
  .selectAll("circle")
  .data(weights)
  .join("circle")
  .attr("class", "hover-target")
  .attr("cx", (d) => x(d.date))
  .attr("cy", (d) => y(d.weight))
  .attr("r", 10)
  .on("mouseenter", (event, d) => showTooltip(event, d))
  .on("mousemove", (event) => positionTooltip(event))
  .on("mouseleave", hideTooltip);

svg.on("mouseleave", hideTooltip);

svg
  .append("text")
  .attr("class", "axis-label")
  .attr("x", margin.left)
  .attr("y", margin.top - 36)
  .text("Weight (lbs)");
