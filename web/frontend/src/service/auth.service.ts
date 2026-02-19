function logout(): Promise<Response> {
    return fetch("/api/logout", {
        method: "PUT",
        credentials: "same-origin",
    });
}

export default {
    logout,
};
