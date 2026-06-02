import { useEffect, useState } from 'react';
import { SkillCard } from './SkillCard';

interface Skill {
  name: string;
  path: string;
}

export function SkillListPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/v1/skills')
      .then(res => res.json())
      .then(data => {
        setSkills(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h2>Skills</h2>
      {skills.length === 0 ? (
        <p>No skills found</p>
      ) : (
        skills.map(skill => <SkillCard key={skill.name} skill={skill} />)
      )}
    </div>
  );
}
