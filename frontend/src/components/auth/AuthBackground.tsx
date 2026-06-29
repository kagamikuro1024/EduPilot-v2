export default function AuthBackground() {
  return (
    <>
      <div className="absolute top-[-10%] left-[-10%] w-96 h-96 bg-purple-300/40 dark:bg-purple-900/20 rounded-full mix-blend-multiply dark:mix-blend-screen filter blur-3xl opacity-70 animate-blob transition-all duration-1000"></div>
      <div className="absolute top-[20%] right-[-10%] w-96 h-96 bg-cyan-300/40 dark:bg-cyan-900/20 rounded-full mix-blend-multiply dark:mix-blend-screen filter blur-3xl opacity-70 animate-blob animation-delay-2000 transition-all duration-1000"></div>
      <div className="absolute bottom-[-20%] left-[20%] w-96 h-96 bg-indigo-300/40 dark:bg-indigo-900/20 rounded-full mix-blend-multiply dark:mix-blend-screen filter blur-3xl opacity-70 animate-blob animation-delay-4000 transition-all duration-1000"></div>
    </>
  );
}
