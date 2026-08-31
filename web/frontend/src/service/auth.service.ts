function logout(): Promise<Response> {
    return fetch("/logout", {
        credentials: "same-origin",
    });
}

export default {
    logout,
};
