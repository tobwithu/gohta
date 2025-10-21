const post = async (url, body) => {
  const response = await fetch(`/api/${url}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  })
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
  async convertFileSrc(filePath) {
    try {
      const result = await post("convertFileSrc", { filePath })
      return result.newSrc
    } catch (error) {
      console.error("Error converting file src:", error)
      return filePath
    }
  },
}