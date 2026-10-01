import { h, render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";

function App() {
  const [info, setInfo] = useState(null);
  const [events, setEvents] = useState([]);
  const [spec, setSpec] = useState(null);
  const [peers, setPeers] = useState([]);
  const [selected, setSelected] = useState(null);
  const [follow, setFollow] = useState(true);
  const [status, setStatus] = useState("connecting");
  const [filters, setFilters] = useState({
    direction: "all",
    query: "",
    method: "",
    id: "",
  });
  const listRef = useRef(null);

  useEffect(() => {
    load();
    const es = new EventSource("/api/events");
    es.onopen = () => setStatus("live");
    es.onerror = () => setStatus("reconnecting");
    es.onmessage = (message) => {
      const payload = JSON.parse(message.data);
      if (payload.type === "event") {
        setEvents((current) => {
          const next = current.concat(payload.event);
          return next.length > 4096 ? next.slice(next.length - 4096) : next;
        });
      }
      if (payload.type === "peers") {
        setPeers(payload.peers || []);
      }
      if (payload.type === "spec") {
        setSpec(payload.spec || null);
      }
    };
    return () => es.close();
  }, []);

  useEffect(() => {
    if (!follow || !listRef.current) {
      return;
    }
    listRef.current.scrollTop = listRef.current.scrollHeight;
  }, [events, follow]);

  useEffect(() => {
    const onKey = (event) => {
      if (!filtered.length) {
        return;
      }
      if (event.key === "j" || event.key === "ArrowDown") {
        event.preventDefault();
        moveSelection(1);
      }
      if (event.key === "k" || event.key === "ArrowUp") {
        event.preventDefault();
        moveSelection(-1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  async function load() {
    const [infoRes, snapshotRes] = await Promise.all([
      fetch("/api/info"),
      fetch("/api/snapshot"),
    ]);
    const infoData = await infoRes.json();
    const snapshotData = await snapshotRes.json();
    setInfo(infoData);
    setPeers(infoData.peers || []);
    setEvents(snapshotData.events || []);
    setSpec(snapshotData.spec || null);
    if ((snapshotData.events || []).length) {
      setSelected(snapshotData.events[snapshotData.events.length - 1].seq);
    }
  }

  async function openPeer(id) {
    const response = await fetch(`/api/peers/open?id=${encodeURIComponent(id)}`, {
      method: "POST",
    });
    if (!response.ok) {
      return;
    }
    const body = await response.json();
    if (body.url) {
      window.location.href = body.url;
    }
  }

  function moveSelection(delta) {
    if (!filtered.length) {
      return;
    }
    const index = filtered.findIndex((event) => event.seq === selected);
    const nextIndex = index < 0 ? filtered.length - 1 : Math.max(0, Math.min(filtered.length - 1, index + delta));
    setSelected(filtered[nextIndex].seq);
  }

  const filtered = useMemo(() => {
    return events.filter((event) => {
      if (filters.direction !== "all" && event.direction !== filters.direction) {
        return false;
      }
      const parsed = parseJSON(event.parsed);
      const method = parsed?.method || "";
      const id = parsed?.id == null ? "" : String(parsed.id);
      const haystack = `${event.formatted} ${method} ${id}`.toLowerCase();
      if (filters.query && !haystack.includes(filters.query.toLowerCase())) {
        return false;
      }
      if (filters.method && !method.toLowerCase().includes(filters.method.toLowerCase())) {
        return false;
      }
      if (filters.id && id !== filters.id) {
        return false;
      }
      return true;
    });
  }, [events, filters]);

  useEffect(() => {
    if (!filtered.length) {
      setSelected(null);
      return;
    }
    const exists = filtered.some((event) => event.seq === selected);
    if (!exists) {
      setSelected(filtered[filtered.length - 1].seq);
    }
  }, [filtered, selected]);

  const active = filtered.find((event) => event.seq === selected) || filtered[filtered.length - 1] || null;
  const detail = active ? parseJSON(active.parsed) : null;
  const activeID = detail?.id == null ? "" : String(detail.id);
  const specDoc = spec?.spec || {};
  const specText = spec?.text || "";
  const specCounts = {
    tools: specDoc.tools?.length || 0,
    resources: specDoc.resources?.length || 0,
    prompts: specDoc.prompts?.length || 0,
  };

  return (
    <div class="shell">
      <aside class="sidebar">
        <div class="brand">
          <div class="brand-mark">mcpspy</div>
          <div class={`status-pill status-${status}`}>{status}</div>
        </div>
        <section class="panel">
          <h2>Peers</h2>
          <div class="peer-list">
            {(peers || []).map((peer) => (
              <button key={peer.id} class="peer" onClick={() => openPeer(peer.id)}>
                <div class="peer-title">{peer.name || peer.command || peer.id}</div>
                <div class="peer-meta">
                  <span>{peer.session_id}</span>
                  <span>{peer.ui_url ? "ui" : "lazy"}</span>
                </div>
              </button>
            ))}
            {(!peers || !peers.length) && <div class="empty">No peers</div>}
          </div>
        </section>
        <section class="panel">
          <h2>Session</h2>
          <dl class="meta-list">
            <dt>Name</dt>
            <dd>{info?.self?.name || "unnamed"}</dd>
            <dt>PID</dt>
            <dd>{info?.self?.pid || "-"}</dd>
            <dt>Command</dt>
            <dd>{info?.self?.command || "-"}</dd>
            <dt>Output</dt>
            <dd>{info?.output_file || "-"}</dd>
          </dl>
        </section>
        <section class="panel">
          <h2>Spec</h2>
          <dl class="meta-list">
            <dt>File</dt>
            <dd>{spec?.path || "-"}</dd>
            <dt>Server</dt>
            <dd>{specDoc.server?.name || info?.self?.name || "unknown"}</dd>
            <dt>Counts</dt>
            <dd>{`${specCounts.tools} tools, ${specCounts.resources} resources, ${specCounts.prompts} prompts`}</dd>
          </dl>
        </section>
      </aside>

      <main class="workspace">
        <header class="toolbar">
          <div class="toolbar-title">
            <strong>{info?.self?.command || "mcpspy"}</strong>
            <span>{info?.url || ""}</span>
          </div>
          <div class="toolbar-controls">
            <select value={filters.direction} onChange={(event) => setFilters((current) => ({ ...current, direction: event.target.value }))}>
              <option value="all">all</option>
              <option value="recv">recv</option>
              <option value="send">send</option>
            </select>
            <input placeholder="search" value={filters.query} onInput={(event) => setFilters((current) => ({ ...current, query: event.target.value }))} />
            <input placeholder="method" value={filters.method} onInput={(event) => setFilters((current) => ({ ...current, method: event.target.value }))} />
            <input placeholder="id" value={filters.id} onInput={(event) => setFilters((current) => ({ ...current, id: event.target.value }))} />
            <label class="toggle">
              <input type="checkbox" checked={follow} onChange={(event) => setFollow(event.target.checked)} />
              <span>follow</span>
            </label>
          </div>
        </header>

        <section class="content">
          <div class="timeline" ref={listRef}>
            {filtered.map((event) => {
              const parsed = parseJSON(event.parsed);
              const method = parsed?.method || responseLabel(parsed);
              const activeRow = event.seq === selected;
              const rowID = parsed?.id == null ? "" : String(parsed.id);
              const linked = activeID && rowID && activeID === rowID;
              return (
                <button key={event.seq} class={`row row-${event.direction} ${activeRow ? "row-active" : ""} ${linked ? "row-linked" : ""}`} onClick={() => setSelected(event.seq)}>
                  <div class="row-top">
                    <span class="row-dir">{event.direction}</span>
                    <span class="row-method">{method || "message"}</span>
                    {rowID && <span class="row-id">id {rowID}</span>}
                    <span class="row-seq">#{event.seq}</span>
                  </div>
                  <div class="row-body">{String(event.formatted)}</div>
                </button>
              );
            })}
            {!filtered.length && <div class="empty empty-main">No matching traffic</div>}
          </div>

          <aside class="detail">
            <div class="detail-stack">
              {!active && <div class="empty">Select a message</div>}
              {active && (
                <div class="detail-card">
                  <div class="detail-head">
                    <div>
                      <div class="detail-title">{active.direction}</div>
                      <div class="detail-subtitle">{new Date(active.time).toLocaleTimeString()}</div>
                    </div>
                    <div class="detail-actions">
                      <button onClick={() => navigator.clipboard.writeText(String(active.formatted))}>copy trace</button>
                      <button onClick={() => navigator.clipboard.writeText(String(active.raw))}>copy json</button>
                    </div>
                  </div>
                  <div class="detail-section">
                    <h3>Trace</h3>
                    <pre>{String(active.formatted)}</pre>
                  </div>
                  <div class="detail-section">
                    <h3>JSON</h3>
                    <pre>{detail ? JSON.stringify(detail, null, 2) : String(active.raw)}</pre>
                  </div>
                  {activeID && (
                    <div class="detail-section">
                      <h3>Correlation</h3>
                      <pre>{filtered.filter((event) => {
                        const parsed = parseJSON(event.parsed);
                        return parsed?.id != null && String(parsed.id) === activeID;
                      }).map((event) => `${event.direction} #${event.seq}`).join("\n")}</pre>
                    </div>
                  )}
                </div>
              )}

              <div class="detail-card spec-card">
                <div class="detail-head">
                  <div>
                    <div class="detail-title">live .mcpspec</div>
                    <div class="detail-subtitle">{spec?.path || "not persisted yet"}</div>
                  </div>
                  <div class="detail-actions">
                    <button onClick={() => navigator.clipboard.writeText(specText || "{}")}>copy spec</button>
                  </div>
                </div>
                <div class="detail-section">
                  <h3>JSON</h3>
                  <pre>{specText || JSON.stringify(specDoc, null, 2)}</pre>
                </div>
              </div>
            </div>
          </aside>
        </section>

        <footer class="footer">
          <span>{filtered.length} visible</span>
          <span>{events.length} buffered</span>
          <span>{`${specCounts.tools}/${specCounts.resources}/${specCounts.prompts}`}</span>
          <span>{info?.self?.session_id || ""}</span>
        </footer>
      </main>
    </div>
  );
}

function parseJSON(value) {
  if (!value) {
    return null;
  }
  try {
    if (typeof value === "string") {
      return JSON.parse(value);
    }
    if (typeof value === "object") {
      return value;
    }
  } catch (_) {
    return null;
  }
  return null;
}

function responseLabel(parsed) {
  if (!parsed) {
    return "";
  }
  if (parsed.method) {
    return parsed.method;
  }
  if (parsed.result !== undefined) {
    return "result";
  }
  if (parsed.error !== undefined) {
    return "error";
  }
  return "message";
}

render(<App />, document.getElementById("app"));
