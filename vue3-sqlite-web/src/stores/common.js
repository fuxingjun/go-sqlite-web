import { ref } from "vue";

function useTokenStore() {
  let token = localStorage.getItem("token");

  function setToken(data) {
    token = data;
    localStorage.setItem("token", data);
  }

  function getToken() {
    return token;
  }

  return { token, setToken, getToken, };
}

export const tokenStore = useTokenStore()
