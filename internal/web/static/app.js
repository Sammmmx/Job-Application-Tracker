const STATUSES = ["Applied", "Interview", "Offer", "Rejected", "Withdrawn"];

const state = {
  items: [],
  total: 0,
  limit: 20,
  offset: 0,
  editingId: null,
  deletingId: null,
};

const els = {
  healthBadge: document.getElementById("healthBadge"),
  addBtn: document.getElementById("addBtn"),
  filterForm: document.getElementById("filterForm"),
  clearFiltersBtn: document.getElementById("clearFiltersBtn"),
  filterCompany: document.getElementById("filterCompany"),
  filterRole: document.getElementById("filterRole"),
  filterStatus: document.getElementById("filterStatus"),
  filterJobType: document.getElementById("filterJobType"),
  filterStartDate: document.getElementById("filterStartDate"),
  filterEndDate: document.getElementById("filterEndDate"),
  banner: document.getElementById("banner"),
  applicationsBody: document.getElementById("applicationsBody"),
  paginationInfo: document.getElementById("paginationInfo"),
  limitSelect: document.getElementById("limitSelect"),
  prevPageBtn: document.getElementById("prevPageBtn"),
  nextPageBtn: document.getElementById("nextPageBtn"),
  statTotal: document.getElementById("statTotal"),
  statApplied: document.getElementById("statApplied"),
  statInterview: document.getElementById("statInterview"),
  statOffer: document.getElementById("statOffer"),
  appModal: document.getElementById("appModal"),
  modalTitle: document.getElementById("modalTitle"),
  closeModalBtn: document.getElementById("closeModalBtn"),
  cancelModalBtn: document.getElementById("cancelModalBtn"),
  appForm: document.getElementById("appForm"),
  formError: document.getElementById("formError"),
  companyInput: document.getElementById("companyInput"),
  roleInput: document.getElementById("roleInput"),
  statusInput: document.getElementById("statusInput"),
  jobTypeInput: document.getElementById("jobTypeInput"),
  jobUrlInput: document.getElementById("jobUrlInput"),
  appliedAtInput: document.getElementById("appliedAtInput"),
  interviewAtInput: document.getElementById("interviewAtInput"),
  notesInput: document.getElementById("notesInput"),
  saveBtn: document.getElementById("saveBtn"),
  deleteModal: document.getElementById("deleteModal"),
  deletePrompt: document.getElementById("deletePrompt"),
  closeDeleteBtn: document.getElementById("closeDeleteBtn"),
  cancelDeleteBtn: document.getElementById("cancelDeleteBtn"),
  confirmDeleteBtn: document.getElementById("confirmDeleteBtn"),
};

async function apiFetch(url, options = {}) {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`);
  }
  return data;
}

function showBanner(msg) {
  els.banner.textContent = msg;
  els.banner.classList.remove("hidden");
}

function hideBanner() {
  els.banner.classList.add("hidden");
}

function formatDate(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function formatDateTime(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function toLocalInputValue(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function localInputToRFC3339(val) {
  if (!val) return null;
  const d = new Date(val);
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}

async function checkHealth() {
  try {
    const data = await apiFetch("/health");
    if (data && data.status === "ok") {
      els.healthBadge.textContent = "API & DB Connected";
      els.healthBadge.className = "health-badge ok";
      return;
    }
  } catch (_) {}
  els.healthBadge.textContent = "API Unavailable";
  els.healthBadge.className = "health-badge down";
}

function buildListQuery() {
  const params = new URLSearchParams();
  params.set("limit", String(state.limit));
  params.set("offset", String(state.offset));
  if (els.filterCompany.value.trim()) params.set("company", els.filterCompany.value.trim());
  if (els.filterRole.value.trim()) params.set("role", els.filterRole.value.trim());
  if (els.filterStatus.value) params.set("status", els.filterStatus.value);
  if (els.filterJobType.value) params.set("job_type", els.filterJobType.value);
  if (els.filterStartDate.value) params.set("start_date", els.filterStartDate.value);
  if (els.filterEndDate.value) params.set("end_date", els.filterEndDate.value);
  return `/applications?${params.toString()}`;
}

async function loadApplications() {
  hideBanner();
  try {
    const data = await apiFetch(buildListQuery());
    state.items = data.items || [];
    state.total = data.total || 0;
    renderStats();
    renderTable();
    renderPagination();
  } catch (err) {
    showBanner(err.message);
    els.applicationsBody.innerHTML = `<tr><td colspan="8" class="empty-cell">Failed to load applications.</td></tr>`;
  }
}

function renderStats() {
  els.statTotal.textContent = state.total;
  const counts = { Applied: 0, Interview: 0, Offer: 0 };
  for (const item of state.items) {
    if (counts[item.status] !== undefined) counts[item.status]++;
  }
  els.statApplied.textContent = counts.Applied;
  els.statInterview.textContent = counts.Interview;
  els.statOffer.textContent = counts.Offer;
}

function renderTable() {
  if (state.items.length === 0) {
    els.applicationsBody.innerHTML = `<tr><td colspan="8" class="empty-cell">No applications found. Click "+ Add Application" to create one.</td></tr>`;
    return;
  }

  els.applicationsBody.innerHTML = "";
  for (const app of state.items) {
    const tr = document.createElement("tr");

    const companyTd = document.createElement("td");
    companyTd.className = "company-name";
    companyTd.textContent = app.company;

    const roleTd = document.createElement("td");
    if (app.job_url) {
      const a = document.createElement("a");
      a.href = app.job_url;
      a.target = "_blank";
      a.rel = "noopener noreferrer";
      a.className = "role-link";
      a.textContent = app.role;
      roleTd.appendChild(a);
    } else {
      roleTd.textContent = app.role;
    }

    const typeTd = document.createElement("td");
    typeTd.textContent = app.job_type || "—";

    const statusTd = document.createElement("td");
    const select = document.createElement("select");
    select.className = `status-select status-${app.status}`;
    select.setAttribute("aria-label", `Status for ${app.company} ${app.role}`);
    for (const s of STATUSES) {
      const opt = document.createElement("option");
      opt.value = s;
      opt.textContent = s;
      opt.selected = s === app.status;
      select.appendChild(opt);
    }
    select.addEventListener("change", async () => {
      const prev = app.status;
      try {
        await apiFetch(`/applications/${app.id}`, {
          method: "PATCH",
          body: JSON.stringify({ status: select.value }),
        });
        await loadApplications();
      } catch (err) {
        select.value = prev;
        showBanner(err.message);
      }
    });
    statusTd.appendChild(select);

    const appliedTd = document.createElement("td");
    appliedTd.textContent = formatDate(app.applied_at);

    const interviewTd = document.createElement("td");
    interviewTd.textContent = formatDateTime(app.interview_at);

    const notesTd = document.createElement("td");
    notesTd.className = "notes-cell";
    notesTd.title = app.notes || "";
    notesTd.textContent = app.notes || "—";

    const actionsTd = document.createElement("td");
    actionsTd.className = "col-actions";
    const wrap = document.createElement("div");
    wrap.className = "row-actions";

    const editBtn = document.createElement("button");
    editBtn.className = "btn btn-ghost btn-sm";
    editBtn.textContent = "Edit";
    editBtn.addEventListener("click", () => openEditModal(app));

    const delBtn = document.createElement("button");
    delBtn.className = "btn btn-ghost btn-sm";
    delBtn.textContent = "Delete";
    delBtn.addEventListener("click", () => openDeleteModal(app));

    wrap.append(editBtn, delBtn);
    actionsTd.appendChild(wrap);

    tr.append(companyTd, roleTd, typeTd, statusTd, appliedTd, interviewTd, notesTd, actionsTd);
    els.applicationsBody.appendChild(tr);
  }
}

function renderPagination() {
  if (state.total === 0) {
    els.paginationInfo.textContent = "Showing 0 of 0";
  } else {
    const start = state.offset + 1;
    const end = Math.min(state.offset + state.items.length, state.total);
    els.paginationInfo.textContent = `Showing ${start}–${end} of ${state.total}`;
  }
  els.prevPageBtn.disabled = state.offset === 0;
  els.nextPageBtn.disabled = state.offset + state.limit >= state.total;
}

function openCreateModal() {
  state.editingId = null;
  els.modalTitle.textContent = "Add Application";
  els.saveBtn.textContent = "Create Application";
  els.formError.classList.add("hidden");
  els.appForm.reset();
  els.statusInput.value = "Applied";
  els.appModal.classList.remove("hidden");
  els.companyInput.focus();
}

function openEditModal(app) {
  state.editingId = app.id;
  els.modalTitle.textContent = "Edit Application";
  els.saveBtn.textContent = "Save Changes";
  els.formError.classList.add("hidden");
  els.companyInput.value = app.company || "";
  els.roleInput.value = app.role || "";
  els.statusInput.value = app.status || "Applied";
  els.jobTypeInput.value = app.job_type || "";
  els.jobUrlInput.value = app.job_url || "";
  els.appliedAtInput.value = toLocalInputValue(app.applied_at);
  els.interviewAtInput.value = toLocalInputValue(app.interview_at);
  els.notesInput.value = app.notes || "";
  els.appModal.classList.remove("hidden");
  els.companyInput.focus();
}

function closeAppModal() {
  els.appModal.classList.add("hidden");
}

function openDeleteModal(app) {
  state.deletingId = app.id;
  els.deletePrompt.textContent = `Delete "${app.role}" at "${app.company}"? This cannot be undone.`;
  els.deleteModal.classList.remove("hidden");
}

function closeDeleteModal() {
  state.deletingId = null;
  els.deleteModal.classList.add("hidden");
}

els.appForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  els.formError.classList.add("hidden");

  const payload = {
    company: els.companyInput.value,
    role: els.roleInput.value,
    status: els.statusInput.value,
    job_type: els.jobTypeInput.value,
    job_url: els.jobUrlInput.value,
    notes: els.notesInput.value,
  };

  const appliedISO = localInputToRFC3339(els.appliedAtInput.value);
  if (appliedISO) {
    payload.applied_at = appliedISO;
  }

  const interviewISO = localInputToRFC3339(els.interviewAtInput.value);
  if (state.editingId !== null) {
    // On PATCH, explicitly send null when cleared so the backend clears interview_at.
    payload.interview_at = interviewISO;
  } else if (interviewISO) {
    payload.interview_at = interviewISO;
  }

  try {
    els.saveBtn.disabled = true;
    if (state.editingId === null) {
      await apiFetch("/applications", { method: "POST", body: JSON.stringify(payload) });
      state.offset = 0;
    } else {
      await apiFetch(`/applications/${state.editingId}`, { method: "PATCH", body: JSON.stringify(payload) });
    }
    closeAppModal();
    await loadApplications();
  } catch (err) {
    els.formError.textContent = err.message;
    els.formError.classList.remove("hidden");
  } finally {
    els.saveBtn.disabled = false;
  }
});

els.confirmDeleteBtn.addEventListener("click", async () => {
  if (state.deletingId === null) return;
  try {
    els.confirmDeleteBtn.disabled = true;
    await apiFetch(`/applications/${state.deletingId}`, { method: "DELETE" });
    closeDeleteModal();
    if (state.items.length === 1 && state.offset > 0) {
      state.offset = Math.max(0, state.offset - state.limit);
    }
    await loadApplications();
  } catch (err) {
    closeDeleteModal();
    showBanner(err.message);
  } finally {
    els.confirmDeleteBtn.disabled = false;
  }
});

els.addBtn.addEventListener("click", openCreateModal);
els.closeModalBtn.addEventListener("click", closeAppModal);
els.cancelModalBtn.addEventListener("click", closeAppModal);
els.closeDeleteBtn.addEventListener("click", closeDeleteModal);
els.cancelDeleteBtn.addEventListener("click", closeDeleteModal);

els.filterForm.addEventListener("submit", (e) => {
  e.preventDefault();
  state.offset = 0;
  loadApplications();
});

els.clearFiltersBtn.addEventListener("click", () => {
  els.filterForm.reset();
  state.offset = 0;
  loadApplications();
});

els.limitSelect.addEventListener("change", () => {
  state.limit = Number(els.limitSelect.value);
  state.offset = 0;
  loadApplications();
});

els.prevPageBtn.addEventListener("click", () => {
  state.offset = Math.max(0, state.offset - state.limit);
  loadApplications();
});

els.nextPageBtn.addEventListener("click", () => {
  state.offset += state.limit;
  loadApplications();
});

checkHealth();
loadApplications();
