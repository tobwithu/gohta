const post = async (url, body = {}) => {
  const response = await fetch(`/api/${url}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return response.json()
}

const get = async (url) => {
  const response = await fetch(`/api/${url}`)
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return response.json()
}

const gohta = {
  async log(message) {
    try {
      await post("log", { message })
    } catch (error) {
      console.error("Error while logging message:", error)
    }
  },
  core:{
    async convertFileSrc(filePath) {
        try {
          const result = await post("core/convertFileSrc", { filePath })
          return result
        } catch (error) {
          console.error("Error converting file src:", error)
          return filePath
        }
      },
    async getArgs(){
      const result = await get("core/getArgs")
      return result
    }
  }
}