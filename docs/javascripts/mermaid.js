mermaid.initialize({ startOnLoad: false });

let mermaidRendering = false;

document$.subscribe(() => {
  if (mermaidRendering) {
    return;
  }

  const nodes = [];
  for (const node of document.querySelectorAll(".mermaid")) {
    if (node.querySelector("svg")) {
      continue;
    }

    const source = node.querySelector(":scope > code")?.textContent ?? node.textContent;
    if (!source.trim()) {
      continue;
    }

    if (node.matches("pre")) {
      const replacement = document.createElement("div");
      replacement.className = "mermaid";
      replacement.textContent = source;
      node.replaceWith(replacement);
      nodes.push(replacement);
      continue;
    }

    nodes.push(node);
  }
  if (nodes.length === 0) {
    return;
  }

  mermaidRendering = true;
  mermaid
    .run({ nodes })
    .catch((error) => {
      console.error("Mermaid rendering failed", error);
    })
    .finally(() => {
      mermaidRendering = false;
    });
});
