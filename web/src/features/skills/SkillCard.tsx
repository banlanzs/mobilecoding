interface Skill {
  name: string;
  path: string;
}

export function SkillCard({ skill }: { skill: Skill }) {
  return (
    <div className="skill-card">
      <h3>{skill.name}</h3>
      <p>{skill.path}</p>
    </div>
  );
}
