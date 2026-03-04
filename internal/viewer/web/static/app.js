(function () {
    "use strict";

    const CHUNK_SIZE = 200;
    const GAP_CHUNK = 250;

    // Directory mode: read ?file= param from URL
    const urlParams = new URLSearchParams(window.location.search);
    const fileParam = urlParams.get("file");
    const fileSuffix = fileParam ? "&file=" + encodeURIComponent(fileParam) : "";
    const fileQuery = fileParam ? "?file=" + encodeURIComponent(fileParam) : "";

    let totalLines = 0;
    let loadedRanges = []; // [{start, end}]
    let comments = [];
    let loading = false;

    const fileContent = document.getElementById("file-content");
    const commentContent = document.getElementById("comment-content");
    const fileInfo = document.getElementById("file-info");

    async function fetchLines(start, end) {
        const resp = await fetch(`/api/lines?start=${start}&end=${end}${fileSuffix}`);
        return resp.json();
    }

    async function fetchComments() {
        const resp = await fetch("/api/comments" + fileQuery);
        return resp.json();
    }

    function isRangeLoaded(start, end) {
        return loadedRanges.some(r => r.start <= start && r.end >= end);
    }

    function mergeRange(start, end) {
        loadedRanges.push({ start, end });
        loadedRanges.sort((a, b) => a.start - b.start);
        const merged = [loadedRanges[0]];
        for (let i = 1; i < loadedRanges.length; i++) {
            const last = merged[merged.length - 1];
            if (loadedRanges[i].start <= last.end + 1) {
                last.end = Math.max(last.end, loadedRanges[i].end);
            } else {
                merged.push(loadedRanges[i]);
            }
        }
        loadedRanges = merged;
    }

    function createLineElement(num, text) {
        const div = document.createElement("div");
        div.className = "line";
        div.dataset.line = num;

        const numSpan = document.createElement("span");
        numSpan.className = "line-number";
        numSpan.textContent = num;

        const textSpan = document.createElement("span");
        textSpan.className = "line-text";
        textSpan.textContent = text;

        div.appendChild(numSpan);
        div.appendChild(textSpan);
        return div;
    }

    function insertLines(data) {
        const lines = data.lines;
        if (!lines || lines.length === 0) return;

        const existingEls = Array.from(fileContent.querySelectorAll(".line"));
        let existingIdx = 0;

        for (const line of lines) {
            while (existingIdx < existingEls.length &&
                   parseInt(existingEls[existingIdx].dataset.line) < line.number) {
                existingIdx++;
            }

            if (existingIdx < existingEls.length &&
                parseInt(existingEls[existingIdx].dataset.line) === line.number) {
                continue;
            }

            const el = createLineElement(line.number, line.text);
            if (existingIdx < existingEls.length) {
                fileContent.insertBefore(el, existingEls[existingIdx]);
            } else {
                fileContent.appendChild(el);
            }
        }
    }

    // --- Gap separators ---

    function createGapSeparator(gapStart, gapEnd) {
        const count = gapEnd - gapStart + 1;
        const div = document.createElement("div");
        div.className = "gap-separator";

        const label = document.createElement("span");
        label.className = "gap-label";
        label.textContent = `\u00b7\u00b7\u00b7 ${count.toLocaleString()} line${count !== 1 ? "s" : ""} not loaded (${gapStart}\u2013${gapEnd}) \u00b7\u00b7\u00b7`;

        const actions = document.createElement("span");
        actions.className = "gap-actions";

        const downBtn = document.createElement("button");
        downBtn.className = "gap-btn";
        downBtn.textContent = "\u2193 load down\u2026";
        downBtn.onclick = () => loadGapChunk(gapStart, Math.min(gapEnd, gapStart + GAP_CHUNK - 1));

        const upBtn = document.createElement("button");
        upBtn.className = "gap-btn";
        upBtn.textContent = "\u2191 load up\u2026";
        upBtn.onclick = () => loadGapChunk(Math.max(gapStart, gapEnd - GAP_CHUNK + 1), gapEnd);

        actions.appendChild(downBtn);
        actions.appendChild(upBtn);
        div.appendChild(label);
        div.appendChild(actions);
        return div;
    }

    function updateGapSeparators() {
        fileContent.querySelectorAll(".gap-separator").forEach(el => el.remove());

        const lines = Array.from(fileContent.querySelectorAll(".line"));
        if (lines.length === 0 || totalLines === 0) return;

        const firstNum = parseInt(lines[0].dataset.line);
        const lastNum = parseInt(lines[lines.length - 1].dataset.line);

        // Top-of-file gap
        if (firstNum > 1) {
            const sep = createGapSeparator(1, firstNum - 1);
            fileContent.insertBefore(sep, lines[0]);
        }

        // Interior gaps
        for (let i = 1; i < lines.length; i++) {
            const prevNum = parseInt(lines[i - 1].dataset.line);
            const currNum = parseInt(lines[i].dataset.line);
            if (currNum > prevNum + 1) {
                const sep = createGapSeparator(prevNum + 1, currNum - 1);
                fileContent.insertBefore(sep, lines[i]);
            }
        }

        // Bottom-of-file gap
        if (lastNum < totalLines) {
            const sep = createGapSeparator(lastNum + 1, totalLines);
            fileContent.appendChild(sep);
        }
    }

    async function loadGapChunk(start, end) {
        if (loading) return;
        loading = true;
        try {
            const data = await fetchLines(start, end);
            totalLines = data.total_lines;
            insertLines(data);
            mergeRange(start, end);
            updateGapSeparators();
        } finally {
            loading = false;
        }
    }

    // --- End gap separators ---

    function highlightCommentLines(comment) {
        fileContent.querySelectorAll(".line.highlighted").forEach(el => {
            el.classList.remove("highlighted");
        });

        if (!comment) return;

        for (let i = comment.range_start; i <= comment.range_end; i++) {
            const el = fileContent.querySelector(`.line[data-line="${i}"]`);
            if (el) el.classList.add("highlighted");
        }
    }

    function renderComments() {
        commentContent.innerHTML = "";

        if (comments.length === 0) {
            commentContent.innerHTML = '<div class="loading">No comments found.</div>';
            return;
        }

        for (const c of comments) {
            const block = document.createElement("div");
            block.className = "comment-block" + (c.drifted ? " drifted" : "");

            const header = document.createElement("div");
            header.className = "comment-header";

            const left = document.createElement("span");
            const idSpan = document.createElement("span");
            idSpan.className = "comment-id";
            idSpan.textContent = c.id;

            const rangeSpan = document.createElement("span");
            rangeSpan.className = "comment-range";
            rangeSpan.textContent = ` L${c.range_start}-${c.range_end}`;
            rangeSpan.onclick = () => scrollToLine(c.range_start);

            left.appendChild(idSpan);
            left.appendChild(rangeSpan);
            header.appendChild(left);

            if (c.drifted) {
                const badge = document.createElement("span");
                badge.className = "drift-badge";
                badge.textContent = "DRIFTED";
                badge.title = "Source content has changed since this comment was created";
                header.appendChild(badge);
            }

            const body = document.createElement("div");
            body.className = "comment-body";
            body.textContent = c.body;

            block.appendChild(header);
            block.appendChild(body);

            block.addEventListener("mouseenter", () => {
                highlightCommentLines(c);
                ensureLinesLoaded(c.range_start, c.range_end);
            });
            block.addEventListener("mouseleave", () => highlightCommentLines(null));

            commentContent.appendChild(block);
        }
    }

    async function ensureLinesLoaded(start, end) {
        if (isRangeLoaded(start, end)) return;

        const chunkStart = Math.max(1, start - CHUNK_SIZE);
        const chunkEnd = Math.min(totalLines, end + CHUNK_SIZE);

        if (loading) return;
        loading = true;

        try {
            const data = await fetchLines(chunkStart, chunkEnd);
            totalLines = data.total_lines;
            insertLines(data);
            mergeRange(chunkStart, chunkEnd);
            updateGapSeparators();
        } finally {
            loading = false;
        }
    }

    async function scrollToLine(lineNum) {
        await ensureLinesLoaded(lineNum - 10, lineNum + 50);
        const el = fileContent.querySelector(`.line[data-line="${lineNum}"]`);
        if (el) {
            el.scrollIntoView({ behavior: "smooth", block: "center" });
        }
    }

    async function loadMoreLines() {
        if (loading) return;

        const scrollTop = fileContent.scrollTop;
        const scrollHeight = fileContent.scrollHeight;
        const clientHeight = fileContent.clientHeight;

        // Load more when near bottom
        if (scrollHeight - scrollTop - clientHeight < 200) {
            const lastLine = fileContent.querySelector(".line:last-of-type");
            if (!lastLine) return;
            const lastNum = parseInt(lastLine.dataset.line);
            if (lastNum >= totalLines) return;

            const start = lastNum + 1;
            const end = Math.min(totalLines, start + CHUNK_SIZE - 1);
            await ensureLinesLoaded(start, end);
        }

        // Load more when near top
        if (scrollTop < 200) {
            const firstLine = fileContent.querySelector(".line");
            if (!firstLine) return;
            const firstNum = parseInt(firstLine.dataset.line);
            if (firstNum <= 1) return;

            const end = firstNum - 1;
            const start = Math.max(1, end - CHUNK_SIZE + 1);
            await ensureLinesLoaded(start, end);
        }
    }

    fileContent.addEventListener("scroll", () => {
        requestAnimationFrame(loadMoreLines);
    });

    // Initial load
    async function init() {
        fileContent.innerHTML = '<div class="loading">Loading...</div>';
        commentContent.innerHTML = '<div class="loading">Loading...</div>';

        try {
            const [linesData, commentsData] = await Promise.all([
                fetchLines(1, CHUNK_SIZE),
                fetchComments(),
            ]);

            totalLines = linesData.total_lines;
            const displayName = fileParam || commentsData.source_file;
            fileInfo.textContent = `${displayName} — ${totalLines} lines, ${commentsData.comments.length} comments`;

            fileContent.innerHTML = "";
            insertLines(linesData);
            mergeRange(1, Math.min(CHUNK_SIZE, totalLines));

            comments = commentsData.comments;
            renderComments();
            updateGapSeparators();

            // In directory mode, add back link and download button before version
            const header = document.getElementById("header");
            const versionEl = document.getElementById("version");
            if (fileParam) {
                const backLink = document.createElement("a");
                backLink.href = "/";
                backLink.textContent = "\u2190 back to directory";
                backLink.style.cssText = "color:#89b4fa;text-decoration:none;margin-left:auto;font-size:12px;";
                backLink.addEventListener("mouseenter", function() { backLink.style.textDecoration = "underline"; });
                backLink.addEventListener("mouseleave", function() { backLink.style.textDecoration = "none"; });
                header.insertBefore(backLink, versionEl);
            }

            // Set up pane-header download links
            const fileDl = document.getElementById("file-download");
            fileDl.href = `/api/download${fileQuery}`;
            fileDl.textContent = "\u2193 download";
            fileDl.style.display = "inline";

            if (comments.length > 0) {
                const commentsDl = document.getElementById("comments-download");
                commentsDl.href = `/api/download-comments${fileQuery}`;
                commentsDl.textContent = "\u2193 download";
                commentsDl.style.display = "inline";
            }

            fetch("/api/version").then(r => r.text()).then(v => {
                versionEl.textContent = v;
            });
        } catch (err) {
            fileContent.innerHTML = `<div class="loading">Error: ${err.message}</div>`;
        }
    }

    init();
})();
